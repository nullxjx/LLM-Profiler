## LLM-Profiler
LLM-Profiler 是一个测试 llm 性能（速度和吞吐量）的工具，适配了 [TensorRT-LLM](https://github.com/NVIDIA/TensorRT-LLM)、[vLLM](https://github.com/vllm-project/vllm/)、[TGI](https://github.com/huggingface/text-generation-inference) 等常见的 LLM 推理框架。
与 [vLLM](https://github.com/vllm-project/vllm/tree/main/benchmarks) 等推理框架的性能测试不同，这些推理框架在测试性能测时候，主要测试的是离线场景下系统的极限吞吐量，就是能压榨出来的系统吞吐量上限，这样比较适合跑 benchmark，用于比较它们之间性能差异。

本工具注重实际在线推理场景下，考虑业务延迟要求、符合线上实际请求分布下的系统吞吐量。 所以并不会像这些推理框架的测试方法一样预先准备特定 batch 大小的数据。测试数据长度的分布也具有一定的离散性，符合在线推理数据分布特点。 同时，工具统计的一些[指标](perf/throughput/statistics.go)也比较符合业务实际的需求。

## 使用说明
### 本地运行

1. **单条速度**测试 (不关注并发)
   - ```go run main.go speed -b vllm -i 127.0.0.1 -p 8100 -m llama-70b -u nullxjx -l 1000``` 
   - -l 参数用于指定输入prompt长度（token数量），不指定的话使用默认很短的prompt
2. **极限吞吐量**测试 (不关注延迟)
   - ```go run main.go peak -b vllm -i 127.0.0.1 -p 8100 -m llama-70b -u nullxjx```
3. **给定延迟吞吐量**测试
   - ```go run main.go inline -b vllm -i 127.0.0.1 -p 8100 -m llama-70b -u nullxjx```
4. **流式请求** (例如chat测试)
   - 修改 [config_local.yml](./config/config_local_template.yml)文件，需要把 stream 参数设置为 true
   - ```go run main.go chat -c config/config_local.yml```
5. **吞吐量**自定义测试
   - 修改 [config_local.yml](./config/config_local_template.yml)文件
   - ```go run main.go custom -c config/config_local.yml```

### 集群运行
1. 需要修改 [k8s_job.yaml](build/k8s_job.yaml) 文件里面的脚本启动参数（含义同上），如下所示
   - ``` args: [ "auto", "-b", "vllm", "-i", "127.0.0.1", "-p", "8100", "-m", "llama-70b", "-u", "nullxjx" ]```
2. 启动/删除job
   - ```kubectl --kubeconfig ~/Desktop/tke-kubeconfig.yaml -n nullxjx apply -f k8s_job.yaml```
   - ```kubectl --kubeconfig ~/Desktop/tke-kubeconfig.yaml -n nullxjx delete -f k8s_job.yaml```
