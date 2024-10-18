import json
import os
from draw_logic import draw_in_sub_figures, draw_throughput_vs_latency
from utils import compare, download_cos_file_to_json


def cal_input_tokens_per_second():
    result_dir = "./"
    files = os.listdir(result_dir)
    result = []
    original = []
    for file in files:
        if file.split(".")[-1] != "json":
            continue
        with open(result_dir + file, 'r', encoding='utf-8') as f:
            data = json.load(f)
            spent_time_list = [e['timeSpent'] for e in data]
            input_tokens_list = [e['inputTokenLen'] for e in data]
            original.append(
                {"Concurrency": file.split("_")[1], "timeSpent": spent_time_list, "tokens": input_tokens_list})
            result.append({"Concurrency": file.split("_")[1],
                           "Data": float(sum(input_tokens_list)) / float(sum(spent_time_list)) * 1000})

    result = sorted(result, key=lambda x: int(x["Concurrency"]))
    return [e['Data'] for e in result]


def visualize_time_metrics(data_list, label, save_dir):
    result = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        concurrency = []
        avg_latency_server = []
        avg_latency_client = []
        p99 = []
        p90 = []
        p80 = []
        for e in point:
            concurrency.append(e['concurrency'])
            avg_latency_server.append(e['avgTimeServerSide'])
            avg_latency_client.append(e['avgTimeClientSide'])
            p99.append(e['P99'])
            p90.append(e['P90'])
            p80.append(e['P80'])
        metrics = {"Avg Latency Server (ms)": avg_latency_server, "Avg Latency Client (ms)": avg_latency_client,
                   "p99(ms)": p99, "p90": p90, "p80": p80}
        result.append({"concurrency": concurrency, "metrics": metrics, "label": data['label']})
    draw_in_sub_figures(result, label, 2, 3, save_dir)


def visualize_inout_metrics(data_list, label, save_dir):
    result = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        concurrency = []
        input_tokens_per_second = []
        input_len_per_request = []
        output_tokens_per_second = []
        output_len_per_second = []
        for e in point:
            concurrency.append(e['concurrency'])
            input_tokens_per_second.append(e['avgInputTokens'])
            input_len_per_request.append(e['avgInputLen'])
            output_tokens_per_second.append(e['avgOutputTokens'])
            output_len_per_second.append(e['avgOutputLen'])
        metrics = {"avgInputTokens": input_tokens_per_second, "avgInputLen": input_len_per_request,
                   "avgOutputTokens": output_tokens_per_second, "avgOutputLen": output_len_per_second}
        result.append({"concurrency": concurrency, "metrics": metrics, "label": data['label']})
    draw_in_sub_figures(result, label, 2, 2, save_dir)


def visualize_token_metrics(data_list, label, save_dir):
    result = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        concurrency = []
        success_rate = []
        total_count = []
        throughput = []
        input_tokens_per_second = []
        output_tokens_per_second = []
        for e in point:
            concurrency.append(e['concurrency'])
            throughput.append(e['throughput'])
            success_rate.append(float(e['success']) / float(e['total']))
            total_count.append(e['total'])
            input_tokens_per_second.append(e['inputTokensPerSecond'])
            output_tokens_per_second.append(e['outputTokensPerSecond'])
        metrics = {"Inferences/Second": throughput, "Input Tokens/Second": input_tokens_per_second,
                   "Output Tokens/Second": output_tokens_per_second, "Success Rate": success_rate,
                   "Total Requests": total_count}
        result.append({"concurrency": concurrency, "metrics": metrics, "label": data['label']})
    draw_in_sub_figures(result, label, 2, 3, save_dir)


def visualize_final_metrics(data_list, label, save_dir, show=True):
    result = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        concurrency = []
        throughput = []
        avg_latency_client = []
        output_tokens_per_second = []
        for e in point:
            concurrency.append(e['concurrency'])
            throughput.append(e['throughput'])
            output_tokens_per_second.append(e['outputTokensPerSecond'])
            avg_latency_client.append(e['avgTimeClientSide'])
        metrics = {"Inferences/Second": throughput, "Output Tokens/Second": output_tokens_per_second,
                   "Avg Latency Client (ms)": avg_latency_client}
        result.append({"concurrency": concurrency, "metrics": metrics, "label": data['label']})
    draw_in_sub_figures(result, label, 1, 3, save_dir, show)


def visualize_throughput_vs_latency(data_list, label, save_dir):
    result = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        avg_latency_client = []
        output_tokens_per_second = []
        for e in point:
            output_tokens_per_second.append(e['outputTokensPerSecond'])
            avg_latency_client.append(float(e['avgTimeClientSide']))
        result.append({"Output Tokens/Second": output_tokens_per_second,
                       "avgTimeClientSide": avg_latency_client,
                       "label": data['label']})
    print(result)
    draw_throughput_vs_latency(result, label, save_dir)


def merge_datas(data_list, label):
    data_list = sorted(data_list, key=lambda x: int(x['label']))
    throughput = []
    latency = []
    for data in data_list:
        point = data['data']
        point = sorted(point, key=lambda x: int(x["concurrency"]))
        avg_latency_client = []
        output_tokens_per_second = []
        for e in point:
            output_tokens_per_second.append(e['outputTokensPerSecond'])
            avg_latency_client.append(float(e['avgTimeClientSide']))
        throughput.append(output_tokens_per_second)
        latency.append(avg_latency_client)

    assert len(throughput) == len(latency)
    size = len(throughput)
    selected_throughput = []
    selected_latency = []

    # 遍历每一条曲线
    for i in range(size):
        keep_throughput = throughput[i]
        keep_latency = latency[i]
        # 当前曲线每个点跟之前一条曲线所有点比较
        if i > 0:
            keep_throughput = []
            keep_latency = []
            for j in range(len(throughput[i])):
                keep = True
                for e in throughput[i - 1]:
                    # 如果前一条曲线存在比当前曲线高的点，则舍弃当前曲线的点
                    if e > throughput[i][j]:
                        keep = False
                        break
                if keep:
                    keep_throughput.append(throughput[i][j])
                    keep_latency.append(latency[i][j])
        # 当前曲线每个点跟后面一条曲线所有点比较
        if i < size - 1:
            try:
                keep_throughput, keep_latency = compare(keep_throughput, keep_latency,
                                                        throughput[i + 1], latency[i + 1])
            except Exception as e:
                print(e)
        selected_throughput += keep_throughput
        selected_latency += keep_latency
    return {"Output Tokens/Second": selected_throughput, "avgTimeClientSide": selected_latency,
            "label": label}


def visualize_throughput_vs_latency_merge(data_list, label, save_dir):
    draw_throughput_vs_latency([merge_datas(data_list, label)], label, save_dir)


if __name__ == "__main__":
    # # 读取JSON配置文件
    with open('input.json', 'r') as file:
        datas = json.load(file)

    results = []
    for data in datas:
        perf_result = download_cos_file_to_json(data['cos_path'])
        if perf_result == "":
            # 说明没获取到值
            continue
        results.append({"data": perf_result, "label": data['label']})

    # visualize_time_metrics(results, "time metrics", "tmp")
    # visualize_inout_metrics(results, "input-output metrics", "tmp")
    # visualize_token_metrics(results, "token metrics", "tmp")
    visualize_final_metrics(results, "metrics_vs_concurrency", "visual_results")
    # visualize_throughput_vs_latency(results, "throughput_vs_latency", "tmp")
    visualize_throughput_vs_latency_merge(results, "throughput_vs_latency", "visual_results")
