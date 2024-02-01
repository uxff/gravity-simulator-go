# Nvidia-DLI 加速计算基础 —— CUDA C/C++

github: https://github.com/CaptainDuke/Nvidia-DLI

FUNDAMENTALS OF ACCELERATED COMPUTING WITH CUDA C/C++

本项目为 NVIDIA Deep Learning Institute 的[CUDA C/C++课程](https://courses.nvidia.com/courses/course-v1:DLI+C-AC-01+V1-ZH/about) 结课测验，通过此课程，可以获得 [学习证书](https://courses.nvidia.com/certificates/d981cc3c658d4520a6d9510bffb41b4f)


# 加速和优化N体模拟器

[n-body](https://en.wikipedia.org/wiki/N-body_problem) 模拟器可以预测通过引力相互作用的一组物体的个体运动。[01-nbody.cu](./01-nbody.cu) 包含一个简单而有效的 n-body 模拟器，适合用于在三维空间移动的物体。我们可通过向该应用程序传递一个命令行参数以影响系统中的物体数量。

该应用程序现有的 CPU 版能够处理 4096 个物体，在计算系统中物体间的交互次数时，每秒约达 3000 万次。您的任务是：

- 利用 GPU 加速程序，并保持模拟的正确性
- 以迭代方式优化模拟器，以使其每秒计算的交互次数超过 300 亿，同时还能处理 4096 个物体 `(2<<11)`
- 以迭代方式优化模拟器，以使其每秒计算的交互次数超过 3250 亿，同时还能处理约 65000 个物体 `(2<<15)`

```
!nvcc -o nbody 09-nbody/01-nbody.cu
```

```
!./nbody 11 # This argument is passed as `N` in the formula `2<<N`, to determine the number of bodies in the system

```
不要忘记，您可以使用-f标志来强制覆盖现有报告文件，因此在开发过程中无需保留多个报告文件。
```
!nsys profile --stats=true -o nbody-report ./nbody
```