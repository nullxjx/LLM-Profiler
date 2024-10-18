该目录存放了 [ShareGPT_V3_unfiltered_cleaned_split](https://huggingface.co/datasets/anon8231489123/ShareGPT_Vicuna_unfiltered/resolve/main/ShareGPT_V3_unfiltered_cleaned_split.json
) 数据集（该数据集比较大，约642M）使用
[llama_tokenizer](https://belladoreai.github.io/llama-tokenizer-js/example-demo/build/) 进行encode之后，根据token数量进行分组，token数量在500左右（5%以内的差距）的prompt放在了[input_tokens_500.json](ShareGPT_V3_unfiltered_cleaned_split/input_tokens_500.json)文件中，以此类推。

该数据集的目的是为了使用一个比较标准和统一的数据集，测试在使用不同token数目输入数据的情况下，系统吞吐量的变化情况。[vllm](https://github.com/vllm-project/vllm/tree/main/benchmarks) 官方也是使用该数据集进行性能测试的。

当然，不同模型的分词效果不一样，这里只是使用llama模型的tokenizer对输入prompt根据token数目进行大概的分类。