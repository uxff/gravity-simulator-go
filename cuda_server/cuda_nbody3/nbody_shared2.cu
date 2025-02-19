//计算后内存居然没更新！！计算失败！
// ref: https://docs.nvidia.com/cuda/cuda-c-programming-guide/index.html?highlight=gencode#cuda-dynamic-parallelism
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pthread.h>
#include "timer.h"
#include "check.h"
#include <cuda_runtime.h>

#define SOFTENING 1e-9f
#define BLOCK_SIZE 1024
#define BLOCK_STRIDE 32

typedef struct
{
    float x, y, z, vx, vy, vz, mass;
    int id;
} Body;

const char *loadFile = "";
const char *saveFile = "list1.json";

struct SavingThreadParam {
  Body *list;
  Body *dolist;
  int n;
  int state; //1==running, 0==stop
  pthread_t tid;
};

void randomizeBodies(float *data, int n)
{
    for (int i = 0; i < n; i++)
    {
        data[i] = 2.0f * (rand() / (float)RAND_MAX) - 1.0f;
    }
}
void randomizeBodyList(Body *oList, int n)
{
    for (int i = 0; i < n; i++)
    {
        oList[i].x = 2000.0f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
        oList[i].y = 2000.0f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
        oList[i].z = 2000.0f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
        oList[i].vx = 0.2f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
        oList[i].vy = 0.2f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
        oList[i].vz = 0.2f * (2.0*(rand() / (float)RAND_MAX) - 1.0f);
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
	__syncthreads();
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
    __syncthreads();
}

void SaveNBody(Body *oList, int nOrb, const char* filename) {
  if (strcmp(filename, "") == 0) {
    return;
  }
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
  //printf("saved %d\n", nOrb);
}

Body *LoadOrbList(const char* loadFile, int *nOrbLoaded) {
    Body *oList = NULL;
    int nOrb = 0;
    FILE* f = fopen(loadFile, "r");
    if (f == NULL) {
      printf("Error loading file %s!\n", loadFile);
      return NULL;
    }
    // read file and count the orbs
    char buf[256] = "";
    int bracketIndent = 0;
    while (fgets(buf, 256, f) != NULL) {
      for (int i=0; i<256 && buf[i] != '\0'; ++i) {
          bracketIndent += buf[i] == '[' ? 1 : 0;
          bracketIndent -= buf[i] == ']' ? 1 : 0;
          if (buf[i] == '[') {
            nOrb += 1;
          }
      }
    }
    nOrb--;
    printf("according to loadFile, nOrb:%d lastIndent:%d\n", nOrb, bracketIndent);
    if (nOrb <= 0 || bracketIndent != 0) {
      printf("file content error! no orbs loaded\n");
      fclose(f);
      return NULL;
    }
    //oList = (Body*)malloc(nOrb * sizeof(Body));
    cudaMallocHost(&oList, nOrb * sizeof(Body));

    rewind(f);
    bracketIndent = 0;
    int orbIdx = 0;
    char restLine[512] = "";
    while (fgets(buf, 256, f) != NULL) {
      strcat(restLine, buf);
      int lastLeftBracket = -1;
      //printf("the restLine len:%d we will handle:<<%s>>\n", strlen(restLine), restLine);
      for (int i=0; i<512 && restLine[i] != '\0'; ++i) {
          if (restLine[i] == '[') {
            bracketIndent += 1;
            lastLeftBracket = i;
            //printf("find [ at:%d bracketIndent:%d\n", lastLeftBracket, bracketIndent);
          }
          if (restLine[i] == ']') {
            bracketIndent -= 1;
            //printf("find ] at:%d bracketIndent:%d\n", lastRightBracket, bracketIndent);
            if (bracketIndent == 1) {
              // 扫到右括号才开始解析
              sscanf(restLine+lastLeftBracket+1, "%f,%f,%f,%f,%f,%f,%f,%d", 
                &oList[orbIdx].x, &oList[orbIdx].y, &oList[orbIdx].z, &oList[orbIdx].vx, &oList[orbIdx].vy, &oList[orbIdx].vz, &oList[orbIdx].mass, &oList[orbIdx].id);
              ;
              //printf("loaded orb:%e,%e,%e,%e,%e,%e,%e,%d\n", oList[orbIdx].x, oList[orbIdx].y, oList[orbIdx].z, oList[orbIdx].vx, oList[orbIdx].vy, oList[orbIdx].vz, oList[orbIdx].mass, oList[orbIdx].id);
              orbIdx += 1;
            }
          }
      }
      if (bracketIndent == 2) {
        strcpy(restLine, restLine+lastLeftBracket+1);
      } else {
        restLine[0] = '\0';
      }
      //printf("bracketIndent:%d [ at:%d ] at:%d restLine:%s\n", bracketIndent, lastLeftBracket, lastRightBracket, restLine);
    }
    fclose(f);
    *nOrbLoaded = nOrb;
    return oList;
}

void* ThreadSavingOrbList(void* ptr) {
  //OrbList *oList = (OrbList*)ptr;
  SavingThreadParam *param = (SavingThreadParam*)ptr;
  while (param->state == 1) {
    usleep(500000);
    cudaMemcpy((void*)param->list, (void*)param->dolist, param->n*sizeof(Body), cudaMemcpyDeviceToHost);
    SaveNBody(param->list, param->n, saveFile);
  }
  return NULL;
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

    Body *oList = NULL;

    if (strcmp(loadFile, "") == 0) {
    	cudaMallocHost(&oList, nBodies * sizeof(Body));
        // randomizeBodies(buf, 6 * nBodies); // Init pos / vel data
        randomizeBodyList(oList, nBodies); // Init pos / vel data
    } else {
        //int nOrbLoaded = 0;
        oList = LoadOrbList(loadFile, &nBodies);
        if (oList == NULL || nBodies == 0) {
          printf("load orb list failed!\n");
          return 1;
        }
    }

    printf("%d orbs loaded\n", nBodies);

    int nBytes = nBodies * sizeof(Body);

    double totalTime = 0.0;

    int deviceId;
    cudaGetDevice(&deviceId);

    size_t threadsPerBlock = BLOCK_SIZE;
    size_t numberOfBlocks = (nBodies + threadsPerBlock - 1) / threadsPerBlock;

    // float *d_buf;
    Body *doList = NULL;//(Body *)d_buf;
    // cudaMalloc(&d_buf, nBytes);
    cudaMalloc((void**)&doList, nBytes);
    /*
   * This simulation will run for 10 cycles of time, calculating gravitational
   * interaction amongst bodies, and adjusting their positions to reflect.
   */

    cudaMemcpy((void*)doList, (void*)oList, nBytes, cudaMemcpyHostToDevice);

    SavingThreadParam param = {oList, doList, nBodies, 1};
    pthread_create(&param.tid, NULL, ThreadSavingOrbList, &param);

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
        bodyForce<<<numberOfBlocks * BLOCK_STRIDE, threadsPerBlock>>>(doList, dt, nBodies); // compute interbody forces
        /*
        * This position integration cannot occur until this round of `bodyForce` has completed.
        * Also, the next round of `bodyForce` cannot begin until the integration is complete.
        */
        integrate_position<<<nBodies / threadsPerBlock, threadsPerBlock>>>(doList, dt, nBodies);

        if (iter == nIters - 1)
        {
            cudaMemcpy((void*)oList, (void*)doList, nBytes, cudaMemcpyDeviceToHost);
        }

    /*******************************************************************/
    // Do not modify the code in this section.
        const double tElapsed = GetTimer() / 1000.0;
        totalTime += tElapsed;

        if (nIters >= 10 && (iter+1)%(nIters/10) == 0) {
            printf("process:%d/%d, time:%.3f cps:%e estimate remain:%.3fs\n", iter+1, nIters, (double(clock()-timeStart)/CLOCKS_PER_SEC), double(long(nBodies)*long(nBodies)*long(iter+1))/(double(clock()-timeStart)/CLOCKS_PER_SEC), double(nIters-iter-1)/double(iter+1)*(double(clock()-timeStart)/CLOCKS_PER_SEC));
	    cudaDeviceSynchronize(); //调用次数越少越好
            cudaMemcpy((void*)oList, (void*)doList, nBytes, cudaMemcpyDeviceToHost);
            //SaveNBody(oList, nBodies, saveFile);//moved into 
        }


        // should do it in a thread async
    }

    double avgTime = totalTime / (double)(nIters);
    float billionsOfOpsPerSecond = 1e-9 * nBodies * nBodies / avgTime;

    clock_t timeUsed = clock() - timeStart;
    param.state = 0; // stop thread
    cudaDeviceSynchronize(); //调用次数越少越好


#ifdef ASSESS
    checkPerformance((void*)oList, billionsOfOpsPerSecond, salt);
#else
    checkAccuracy((float*)oList, nBodies);
    SaveNBody(oList, nBodies, saveFile);
    printf("%d Bodies: average %0.3f Billion Interactions / second, cps:%e\n", nBodies, billionsOfOpsPerSecond, (double(nBodies)*double(nBodies)*double(nIters))/(double(clock()-timeStart) / CLOCKS_PER_SEC));
    salt += 1;
#endif
    /*******************************************************************/

    /*
   * Feel free to modify code below.
   */
    cudaFree(doList);
    cudaFreeHost(oList);
}
