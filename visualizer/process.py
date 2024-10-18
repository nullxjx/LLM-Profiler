import os
import re
import json
import argparse
from datetime import datetime
from utils import download_from_cos_url, upload_file_to_cos, send_wechat_message, read_envs
from perf_visualizer import visualize_final_metrics, draw_throughput_vs_latency, merge_datas
from pdf_writer import create_peak_pdf, create_inline_pdf


def find_convergence_value(float_list, window_size=3, threshold=0.05):
    convergence_start = None
    for i in range(window_size, len(float_list)):
        window = float_list[i - window_size: i]
        max_value = max(window)
        min_value = min(window)
        if max_value / min_value - 1 <= threshold:
            convergence_start = i - window_size
            break

    if convergence_start is not None:
        convergence_values = float_list[convergence_start:]
        max_value = max(convergence_values)
        min_value = min(convergence_values)
        remaining_values = [value for value in convergence_values if value != max_value and value != min_value]
        return sum(remaining_values) / len(remaining_values)

    return float_list[-1]


def load_from_local_file(save_dir):
    pattern = r'^statistics_\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.json$'

    for filename in os.listdir(save_dir):
        if re.match(pattern, filename):
            file_path = os.path.join(save_dir, filename)
            with open(file_path, "r") as f:
                return json.load(f)


def calculate_max_throughput(original_data, save_mode='local'):
    for input_tokens, value in original_data.items():
        for max_new_tokens, data in value.items():
            json_data = ""
            if save_mode == "cos":
                json_data = download_from_cos_url(data["cos_url"])
            elif save_mode == "local":
                json_data = load_from_local_file(data["cos_url"])
            throughput = []
            for e in json_data:
                throughput.append(e['outputTokensPerSecond'])
            data['throughput'] = round(find_convergence_value(throughput), 3)
    return original_data


def process_peak_results(original_data):
    user = original_data['config']['user']
    save_mode = original_data['config']['save']

    metrics = calculate_max_throughput(original_data['data'], save_mode)
    metrics = {k1: {k2: v2 for k2, v2 in sorted(inner_dict.items(), key=lambda x: int(x[0]))}
               for k1, inner_dict in sorted(metrics.items(), key=lambda x: int(x[0]))}

    throughput_vs_latency = []
    all_images = []
    for input_tokens, value in metrics.items():
        results = []
        for max_new_tokens, e in value.items():
            json_data = ""
            if save_mode == "cos":
                json_data = download_from_cos_url(e["cos_url"])
            elif save_mode == "local":
                json_data = load_from_local_file(e["cos_url"])
            if json_data == "":
                continue
            results.append({"data": json_data, "label": max_new_tokens})
        label = "metrics_vs_concurrency_of_input_tokens_{}".format(input_tokens)
        visualize_final_metrics(results, label, "visual_results", False)
        all_images.append({"title": label, "path": "visual_results/" + label + ".png"})
        throughput_vs_latency.append(merge_datas(results, "input_tokens: {}".format(input_tokens)))
    draw_throughput_vs_latency(throughput_vs_latency, "throughput_vs_latency", "visual_results", False)
    all_images.append({"title": "throughput_vs_latency", "path": "visual_results/throughput_vs_latency.png"})

    now = datetime.now().strftime("%Y-%m-%d_%H:%M:%S")
    file_name = "/workspace/visualizer/model_performance_report_{}.pdf".format(now)
    create_peak_pdf(file_name, "Model Performance Report", metrics, all_images, original_data['config'])
    print("generate report success, see pdf report in {}".format(file_name))

    upload_file_to_cos(file_name, user)


def process_inline_results(original_data):
    config = original_data['config']
    data = original_data['data']
    # 按照 timeoutSeconds, inputTokens, outputTokens 进行升序排序
    metrics = sorted(
        data,
        key=lambda x: (x["timeoutSeconds"], x["inputTokens"], x["outputTokens"]),
    )
    now = datetime.now().strftime("%Y-%m-%d_%H:%M:%S")
    file_name = "/workspace/visualizer/model_performance_report_{}.pdf".format(now)
    create_inline_pdf(file_name, "Model Performance Report", metrics, config)
    print("generate report success, see pdf report in {}".format(file_name))

    upload_file_to_cos(file_name, user)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="generate pdf report from json file")
    parser.add_argument("--json", type=str, default="peak_results.json")
    parser.add_argument("--type", type=str, default="peak")
    args = parser.parse_args()
    user = "nullxjx"

    read_envs()
    try:
        with open(args.json, "r") as f:
            original_data = json.load(f)
            f.close()
        user = original_data['config']['user']
        if args.type == "peak":
            process_peak_results(original_data)
        elif args.type == "inline":
            process_inline_results(original_data)
    except Exception as e:
        msg = ("# 😭 Auto Performance Test Failed\n\n"
               "Reason: {}\n\n<@{}>\n").format(e, user)
        print("generate pdf error: {}".format(e))
        send_wechat_message(msg)
