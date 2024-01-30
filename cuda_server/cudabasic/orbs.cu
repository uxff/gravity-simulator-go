//%%writefile Orbs.cu
//This file is tested successfully on Google Colab with V100 GPU.
//cps:8.269329e+11 in Google Colab with V100 GPU
//cps:1.588200e+13 in Google Colab with A100 GPU
//cps:3.656473e+11 in Google Colab with A100 GPU
//cps:3.549229e+10 in Google Colab with T4 GPU, more size of orbs will run faster.
#include<stdio.h>
#include<stdlib.h>
#include<cuda.h>
#include<time.h>
//#include<math.h> //cuda already has sqrt

struct Orb {
  double x;
  double y;
  double z;
  double vx;
  double vy;
  double vz;
  double mass;
  int id;
};

const double G  = 0.000005;
const double SPEED_LIMIT = 4.0;
const double MIN_DIST = 0.5;
const double PI = 3.14159265358979323846;

__device__ void OrbUpdate(Orb *o, Orb *oList, int nOrb) {
  if (o->id > 0) {
    //double aAll = CalcGravityAll()
    double gAllx = 0, gAlly = 0, gAllz = 0;
    for (int i=0; i<nOrb; ++i) {
      Orb *target = &oList[i];
      if (target->id < 0 || target->id == o->id) {
        return;
      }
      double distSq = (o->x-target->x)*(o->x-target->x) + (o->y-target->y)*(o->y-target->y) + (o->z-target->z)*(o->z-target->z);
      double dist = sqrt(distSq);

      // if tooNearly or overSpeeded
      if (dist < MIN_DIST) {
        o->id = - o->id; // mark status
        // here need transfer mass to target
        target->mass += o->mass; // will cause concurrency problem.
        o->mass = 0.000000001;
        printf("Orb(%d) got crashed by dist\n", o->id);
        return;
      }

      double gTar = target->mass / distSq * G;
      gAllx += -gTar * (o->x-target->x) / dist;
      gAlly += -gTar * (o->y-target->y) / dist;
      gAllz += -gTar * (o->z-target->z) / dist;
    }

    o->x += o->vx;
    o->y += o->vy;
    o->z += o->vz;
    o->vx += gAllx;
    o->vy += gAlly;
    o->vz += gAllz;

    if (o->vx > SPEED_LIMIT || o->vy > SPEED_LIMIT || o->vz > SPEED_LIMIT) {
      o->id = - o->id;
      printf("Orb(%d) get crashed by overspeed\n", o->id);
      return;
    }
  }
}


__device__ void UpdateOrbList(Orb *oList, int nOrb) {
  for (int i=0; i<nOrb; ++i) {
      Orb *o = &oList[i];
      OrbUpdate(o, oList, nOrb);
  }
  // should clear when orb.id < 0
}

__global__ void UpdateOrbs(Orb *oList, int nOrb, int nTimes) {
  for (int i=0; i<nTimes; ++i) {
    UpdateOrbList(oList, nOrb);
  }
}

const double MASS_RANGE = 1;
const double DISTRI_WIDE = 10000;
const double VELO_RANGE = 0.005;

int main()
{
    int nOrb = 10000;
    int nTimes = 10000;
    
    // 申请host内存
    Orb *oList = (Orb*)malloc(nOrb * sizeof(Orb));

    // 初始化数据
    for (int i = 0; i < nOrb; ++i) {
      oList[i].id = i+1;
      oList[i].mass = (double)rand() / RAND_MAX * MASS_RANGE;
      double radius = DISTRI_WIDE * (double)rand() / RAND_MAX;
      double idx = (double)rand() / RAND_MAX * PI * 2;
      oList[i].x = cos(idx) * radius;
      oList[i].y = sin(idx) * radius;
      oList[i].z = (double)rand() / RAND_MAX - 0.5;
      oList[i].vx = cos(idx+PI/2.0) * VELO_RANGE;
      oList[i].vy = sin(idx+PI/2.0) * VELO_RANGE;
      //printf("[%f,%f,%f,%f,%f,%f,%f,%d]\n", oList[i].x, oList[i].y, oList[i].z, oList[i].vz, oList[i].vy, oList[i].vz, oList[i].mass, oList[i].id);
    }

    printf("init ok, nOrb:%d nTimes:%d, will times:%ld\n", nOrb, nTimes, long(nOrb)*long(nOrb)*long(nTimes));
    clock_t timeStart = clock();

    // 申请device内存
    Orb *doList;
    cudaMalloc((void**)&doList, nOrb*sizeof(Orb));

    // 将host数据拷贝到device
    cudaMemcpy((void*)doList, (void*)oList, nOrb*sizeof(Orb), cudaMemcpyHostToDevice);
    // 定义kernel的执行配置
    //dim3 blockSize(256);
    //dim3 gridSize((nOrb + blockSize.x - 1) / blockSize.x);
    // 执行kernel
    //UpdateOrbs <<< gridSize, blockSize >>>(oList, nOrb, nTimes);
    UpdateOrbs <<< 32, 32 >>>(oList, nOrb, nTimes);
    //UpdateOrbs(oList, nOrb, nTimes); // use host

    // 将device得到的结果拷贝到host
    cudaMemcpy((void*)oList, (void*)doList, nOrb*sizeof(Orb), cudaMemcpyDeviceToHost);

    clock_t timeEnd = clock();

    // 检查执行结果
    printf("all done. nOrb:%d times:%ld use time:%f cps:%e\n", 
      nOrb, 
      long(nOrb)*long(nOrb)*long(nTimes), 
      double(timeEnd-timeStart)/CLOCKS_PER_SEC, 
      double(long(nOrb)*long(nOrb)*long(nTimes))/(double(timeEnd-timeStart)/CLOCKS_PER_SEC));

    // 释放device内存
    cudaFree(doList);
    // 释放host内存
    free(oList);

    return 0;
}