import os
import matplotlib.pyplot as plt
import pandas as pd

# 关闭交互模式
plt.ioff()


def moving_average(data, window_size):
    return pd.Series(data).rolling(window=window_size).mean().tolist()


def draw_in_sub_figures(data, title, rows, cols, save_dir=None, show=True):
    """
    在一副图中创建多个子图，绘制不同的内容
    :param data:
    :param title:
    :return:
    """
    # 创建2x3子图网格
    fig, axes = plt.subplots(rows, cols, figsize=(20, 10))
    axes = axes.flatten()

    # 绘制子图
    metrics_keys = data[0]['metrics']
    for i, (metric, ax) in enumerate(zip(metrics_keys, axes)):
        for e in data:
            # smoothed_data = moving_average(e["metrics"][metric], window_size=3)
            smoothed_data = e["metrics"][metric]
            ax.plot(e["concurrency"], smoothed_data, label=e["label"], marker='o')
            # ax.plot(e["concurrency"], e["metrics"][metric], label=e["label"], marker='o')
        # ax.set_title(metric)
        ax.set_xlabel('Concurrency')
        ax.set_ylabel(metric)
        ax.legend()

    # 如果提供了保存目录，则创建目录（如果不存在）
    if save_dir:
        os.makedirs(save_dir, exist_ok=True)
        file_path = os.path.join(save_dir, title + ".png")
    else:
        file_path = "{}.png".format(title)
    # 保存图像到文件
    plt.savefig(file_path, dpi=300)

    if show:
        # 显示图形
        plt.suptitle(title)
        plt.show()


def draw_throughput_vs_latency(data_list, title, save_dir=None, show=True):
    # 清除当前图
    plt.clf()

    # 绘制曲线
    for data in data_list:
        plt.plot(data["avgTimeClientSide"], data["Output Tokens/Second"], label=data["label"], marker='o')

    # 设置坐标轴标签
    plt.xlabel("latency (ms)")
    plt.ylabel("throughput (tokens/s)")

    # 添加图例
    plt.legend()

    # 如果提供了保存目录，则创建目录（如果不存在）
    if save_dir:
        os.makedirs(save_dir, exist_ok=True)
        file_path = os.path.join(save_dir, title + ".png")
    else:
        file_path = "{}.png".format(title)
    # 保存图像到文件
    plt.savefig(file_path, dpi=300)

    if show:
        plt.show()


if __name__ == "__main__":
    # with open("points_4x4.json", 'r', encoding='utf-8') as f:
    #     points1 = json.load(f)
    # with open("points_8x2.json", 'r', encoding='utf-8') as f:
    #     points2 = json.load(f)
    # draw(points1, points2)
    pass
