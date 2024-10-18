import requests
import json
import os
from qcloud_cos import CosConfig
from qcloud_cos import CosS3Client

# 替换为你的密钥和地域
secret_id = ""
secret_key = ""
region = ""
bucket = ""
prefix = '/statistics_'


def read_envs():
    global secret_id, secret_key, region, bucket
    secret_id = os.environ.get("SECRET_ID", 'AKIDXXX')
    secret_key = os.environ.get("SECRET_KEY", 'XXX')
    region = os.environ.get("REGION", 'ap-shanghai')
    bucket = os.environ.get("BUCKET", 'ai-file-xxx')


def download_from_cos_url(cos_url):
    """
    从cos_url下载文件到json中
    :param cos_url:
    :return:
    """
    response = requests.get(cos_url)

    if response.status_code == 200:
        content = response.text
        return json.loads(content)
    else:
        print("download {} failed! status code: {}".format(cos_url, response.status_code))

    return ""


def download_cos_file_to_json(cos_path):
    """
    从cos路径下载指定文件到json中
    :param cos_path:
    :return:
    """
    config = CosConfig(Region=region, Secret_id=secret_id, Secret_key=secret_key)
    client = CosS3Client(config)

    # 列出目录下的所有文件
    response = client.list_objects(Bucket=bucket, Prefix=cos_path + prefix)
    # 查找匹配的文件
    matching_file = None
    for content in response['Contents']:
        if prefix in content['Key']:
            matching_file = content['Key']
            break
    if matching_file:
        url = client.get_presigned_download_url(Bucket=bucket, Key=matching_file)
    else:
        print("cannot find statistics_*.json file")
        return ""

    return download_from_cos_url(url)


def upload_file_to_cos(file_path, user="nullxjx"):
    save_2_cos = os.environ.get("SAVE2COS", "false")
    if not save_2_cos.lower() in ['true', '1', 't', 'y', 'yes']:
        return

    config = CosConfig(Region=region, SecretId=secret_id, SecretKey=secret_key)
    client = CosS3Client(config)

    file_name = os.path.basename(file_path)

    with open(file_path, 'rb') as fp:
        response = client.put_object(
            Bucket=bucket,
            Body=fp,
            Key="perf_analyzer/report/{}".format(file_name),
            EnableMD5=True
        )
        print("result: {}".format(response))

    # 检查上传状态
    if 'ETag' in response:
        # 生成临时访问链接
        signed_url = client.get_presigned_url(
            Bucket=bucket,
            Key="perf_analyzer/report/{}".format(file_name),
            Method="GET",
            Expired=24 * 3600  # 链接有效时间，单位为秒
        )
        print("Upload {} to cos success. access URL: {}".format(file_path, signed_url))

        msg = ("# 🥳🤩🥰 Auto Performance Test Done\n\n"
               "See summary report via [report.pdf]({})\n\n<@{}>\n").format(signed_url, user)
        send_wechat_message(msg)
    else:
        print("Upload failed. Reason: {}".format(response))


def send_wechat_message(content):
    payload = {
        "msgtype": "markdown",
        "markdown": {
            "content": content
        }
    }
    webhook_url = os.environ.get('WEBHOOK_URL')
    response = requests.post(webhook_url, json=payload)
    if response.status_code != 200:
        print("Error sending wechat message")
    else:
        print("Wechat message sent successfully")


def is_close(a, b, tolerance=0.1):
    if a == 0 and b == 0:
        return True
    if a == 0 or b == 0:
        return False
    relative_error = abs((a - b) / max(abs(a), abs(b)))
    return relative_error <= tolerance


def compare(throughput_1, latency_1, throughput_2, latency_2):
    """
    :param throughput_1: 当前曲线的吞吐量
    :param latency_1: 当前曲线的延迟
    :param throughput_2: 后一条曲线的吞吐量
    :param latency_2: 后一条曲线的延迟
    :return:
    """
    keep_throughput = []
    keep_latency = []
    for i in range(len(throughput_1)):
        keep = True
        for j in range(len(throughput_2)):
            if is_close(latency_1[i], latency_2[j]) and throughput_1[i] < throughput_2[j]:
                keep = False
                break
            if (i < len(latency_1) - 1 and latency_1[i] < latency_2[j] < latency_1[i + 1]
                    and throughput_1[i] < throughput_2[j]):
                keep = False
                break
        if keep:
            keep_throughput.append(throughput_1[i])
            keep_latency.append(latency_1[i])
        else:
            return keep_throughput, keep_latency
    return keep_throughput, keep_latency


def test_compare():
    throughput_1 = [
        13.198095359102156,
        19.866464470095693,
        26.461853710112056,
        33.08373774912212,
        39.69206908270938,
        46.26414885304307,
        52.86777490043511,
        59.28974842868571,
        65.57075368132153,
        72.05498824217366,
        76.11969824630431,
        76.66632836925727,
        77.6502714945159,
        76.3856230274404,
        76.2234881975316
    ]
    latency_1 = [
        362.75757575757575,
        361.5838926174497,
        882.1809045226131,
        966.2690763052209,
        1052.1337792642141,
        1194,
        1348.092731829574,
        1754.5812917594656,
        2733.769076305221,
        4035.1020036429873,
        5550.633779264214,
        7668.819722650231,
        9799.838340486409,
        13126.493991989319,
        15659.589486858573
    ]
    throughput_2 = [
        26.338585551859982,
        39.520228515109274,
        52.664750140061315,
        65.81736575583668,
        79.02612220020951,
        92.04555084641129,
        105.18722251417614,
        117.95135070280327,
        128.39678827431126,
        132.38127483838258,
        131.2283125181529,
        132.67318959770387,
        131.31574022446347,
        133.23889387254675,
        133.23852386807025
    ]
    latency_2 = [
        995.2727272727273,
        1504.496644295302,
        1581.7437185929648,
        1747.3975903614457,
        1954.505016722408,
        2262.3610315186247,
        2552.6390977443607,
        3729.389755011136,
        5640.74749498998,
        7947.653916211293,
        11086.42570951586,
        13916.637904468413,
        16820.76967095851,
        20176.704545454544,
        22945.62703379224
    ]
    keep_throughput, keep_latency = compare(throughput_1, latency_1, throughput_2, latency_2)
    print("before: {}, len: {}".format(throughput_1, len(throughput_1)))
    print("after: {}, len: {}".format(keep_throughput, len(keep_throughput)))
