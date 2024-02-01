#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include "timer.h"
#include "check.h"
#include <cuda_runtime.h>

#define SOFTENING 1e-9f
#define BLOCK_SIZE 1024
#define BLOCK_STRIDE 32

typedef struct
{
    float x, y, z, vx, vy, vz;
} Body;

const char *loadFile = "";
const char *saveFile = "list1.json";

void randomizeBodies(float *data, int n)
{
    for (int i = 0; i < n; i++)
    {
        data[i] = 2.0f * (rand() / (float)RAND_MAX) - 1.0f;
    }
}

__global__ void bodyForce(Body *pList, float dt, int n)
{

    // int i = threadIdx.x + blockIdx.x * blockDim.x;
    // 计算要处理的数据index
    int i = threadIdx.x + (int)(blockIdx.x / BLOCK_STRIDE) * blockDim.x;
    // 此块对应要处理的数据块的起始位置
    int start_block = blockIdx.x % BLOCK_STRIDE;
    if (i < n)
    {
        int cycle_times = n / BLOCK_SIZE;
        Body ptemp = pList[i];
        // 使用shared_memory 多个线程读取同一块数据进入，提升存取性能
        __shared__ float3 spos[BLOCK_SIZE];
        Body temp;
        float dx, dy, dz, distSqr, invDist, invDist3;
        float Fx = 0.0f;
        float Fy = 0.0f;
        float Fz = 0.0f;
        // 这里的cycle_times 在已知块大小时使用常数性能会高一些
        for (int block_num = start_block; block_num < cycle_times; block_num += BLOCK_STRIDE)
        {
            temp = pList[block_num * BLOCK_SIZE + threadIdx.x];
            spos[threadIdx.x] = make_float3(temp.x, temp.y, temp.z);
            // 块内同步，防止spos提前被读取
            __syncthreads();
// 编译优化，只有 BLOCK_SIZE 为常量时才有用
#pragma unroll
            for (int j = 0; j < BLOCK_SIZE; j++)
            {
                dx = spos[j].x - ptemp.x;
                dy = spos[j].y - ptemp.y;
                dz = spos[j].z - ptemp.z;
                distSqr = dx * dx + dy * dy + dz * dz + SOFTENING;
                invDist = rsqrtf(distSqr);
                invDist3 = invDist * invDist * invDist;
                Fx += dx * invDist3;
                Fy += dy * invDist3;
                Fz += dz * invDist3;
            }
            // 块内同步，防止spos提前被写入
            __syncthreads();
        }
        // 块之间不同步，原子加保证正确性
        atomicAdd(&pList[i].vx, dt * Fx);
        atomicAdd(&pList[i].vy, dt * Fy);
        atomicAdd(&pList[i].vz, dt * Fz);
        // pList[i].vx += dt * Fx;
        // pList[i].vy += dt * Fy;
        // pList[i].vz += dt * Fz;
    }
}

__global__ void integrate_position(Body *pList, float dt, int n)
{
    int i = threadIdx.x + blockIdx.x * blockDim.x;
    if (i < n)
    {
        pList[i].x += pList[i].vx * dt;
        pList[i].y += pList[i].vy * dt;
        pList[i].z += pList[i].vz * dt;
    }
}

void SaveNBody(Body *oList, int nOrb, const char* filename) {
  FILE* f = fopen(filename, "w");
  if (f == NULL) {
    printf("Error opening file %s!\n", filename);
    return;
  }
  fputs("[", f);
  for (int i=0; i<nOrb; ++i) {
      fprintf(f, "[%.15g,%.15g,%.15g,%.15g,%.15g,%.15g,%g,%d]", oList[i].x, oList[i].y, oList[i].z, oList[i].vx, oList[i].vy, oList[i].vz, 1.0, i+1);
      if (i < nOrb-1) {
        fputs(",", f);
      }
  }
  fputs("]", f);
  fclose(f);
}


int main(const int argc, const char **argv)
{

    int nBodies = 2 << 11;
    int salt = 0;
    if (argc > 1)
        nBodies = 2 << atoi(argv[1]);

    /*
   * This salt is for assessment reasons. Tampering with it will result in automatic failure.
   */

    if (argc > 2)
        salt = atoi(argv[2]);

    const float dt = 0.01f; // time step
    int nIters = 10;  // simulation iterations

    // Parse arguments
    if (argc >= 2) {
      for (int i = 0; i < argc; ++i) {
        if (strcmp(argv[i], "-n") == 0 && i+1 < argc) {
            nBodies = atoi(argv[i+1]);
        }
        if (strcmp(argv[i], "-t") == 0 && i+1 < argc) {
            nIters = atoi(argv[i+1]);
        }
        if (strcmp(argv[i], "-l") == 0 && i+1 < argc) {
            loadFile = argv[i+1];
        }
        if (strcmp(argv[i], "-s") == 0 && i+1 < argc) {
            saveFile = argv[i+1];
        }
      }
    }

    int bytes = nBodies * sizeof(Body);
    float *buf;
    cudaMallocHost(&buf, bytes);

    randomizeBodies(buf, 6 * nBodies); // Init pos / vel data

    double totalTime = 0.0;

    int deviceId;
    cudaGetDevice(&deviceId);

    size_t threadsPerBlock = BLOCK_SIZE;
    size_t numberOfBlocks = (nBodies + threadsPerBlock - 1) / threadsPerBlock;

    float *d_buf;
    cudaMalloc(&d_buf, bytes);
    Body *d_p = (Body *)d_buf;
    /*
   * This simulation will run for 10 cycles of time, calculating gravitational
   * interaction amongst bodies, and adjusting their positions to reflect.
   */

    cudaMemcpy(d_buf, buf, bytes, cudaMemcpyHostToDevice);

    clock_t timeStart = clock();

    /*******************************************************************/
    // Do not modify these 2 lines of code.gg
    for (int iter = 0; iter < nIters; iter++)
    {
        StartTimer();
    /*******************************************************************/

        /*
        * You will likely wish to refactor the work being done in `bodyForce`,
        * as well as the work to integrate the positions.
        */
        bodyForce<<<numberOfBlocks * BLOCK_STRIDE, threadsPerBlock>>>(d_p, dt, nBodies); // compute interbody forces
        /*
        * This position integration cannot occur until this round of `bodyForce` has completed.
        * Also, the next round of `bodyForce` cannot begin until the integration is complete.
        */
        integrate_position<<<nBodies / threadsPerBlock, threadsPerBlock>>>(d_p, dt, nBodies);

        if (iter == nIters - 1)
        {
            cudaMemcpy(buf, d_buf, bytes, cudaMemcpyDeviceToHost);
        }

    /*******************************************************************/
    // Do not modify the code in this section.
        const double tElapsed = GetTimer() / 1000.0;
        totalTime += tElapsed;

        //
    }

    double avgTime = totalTime / (double)(nIters);
    float billionsOfOpsPerSecond = 1e-9 * nBodies * nBodies / avgTime;

    clock_t timeUsed = clock() - timeStart;

#ifdef ASSESS
    checkPerformance(buf, billionsOfOpsPerSecond, salt);
#else
    checkAccuracy(buf, nBodies);
    SaveNBody((Body*)buf, nBodies, "result.txt");
    printf("%d Bodies: average %0.3f Billion Interactions / second, cps:%e\n", nBodies, billionsOfOpsPerSecond, double(timeUsed)/(double(nBodies)*double(nBodies)*double(nIters)) / CLOCKS_PER_SEC);
    salt += 1;
#endif
    /*******************************************************************/

    /*
   * Feel free to modify code below.
   */
    cudaFree(d_buf);
    cudaFreeHost(buf);
}