//%%writefile Orbs.cu
//This file is tested successfully on Google Colab with V100 GPU.
//cps:8.269329e+11 in Google Colab with V100 GPU
//cps:3.656473e+11 in Google Colab with A100 GPU
//cps:3.549229e+10 in Google Colab with T4 GPU, more size of orbs will run faster.
// build: nvcc -o Orbs Orbs.cu -lm -lpthread
#include<stdio.h>
#include<stdlib.h>
#include<cuda.h>
#include<time.h>
#include<unistd.h>
#include<pthread.h>

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

struct OrbList {
  Orb *list;
  int n;
};
struct SavingThreadParam {
  Orb *list;
  int n;
  int state; //1==running, 0==stop
  pthread_t tid;
};

const double PI = 3.14159265358979323846;
const double G  = 0.00005;
const double SPEED_LIMIT = 4.0;
const double MIN_DIST = 0.5;
const double MASS_RANGE = 100;
const double DISTRI_WIDE = 10000;
const double VELO_RANGE = 0.005;

__device__ void OrbUpdate(Orb *o, Orb *oList, int nOrb) {
  if (o->id > 0) {
    double gAllx = 0, gAlly = 0, gAllz = 0; // double3 gAll = {0, 0, 0};
    for (int i=0; i<nOrb; ++i) {
      Orb *target = &oList[i];
      if (target->id < 0 || target->id == o->id) {
        continue;
      }

      double distSq = (o->x-target->x)*(o->x-target->x) + (o->y-target->y)*(o->y-target->y) + (o->z-target->z)*(o->z-target->z);
      // if tooNearly or overSpeeded
      if (distSq < MIN_DIST*MIN_DIST) {
        o->id = - o->id; // mark status
        //target->mass += o->mass; // transfer mass to target, will cause concurrency problem.
        //o->mass = 0.000000001;
        //printf("%d crashed by MIN_DIST\n", o->id);
        break;
      }
      
      double rdist = rsqrt(distSq);
      double gTar = target->mass / distSq * G;
      gAllx += -gTar * (o->x-target->x) * rdist;
      gAlly += -gTar * (o->y-target->y) * rdist;
      gAllz += -gTar * (o->z-target->z) * rdist;
    }
    
    o->x += o->vx;
    o->y += o->vy;
    o->z += o->vz;
    o->vx += gAllx;
    o->vy += gAlly;
    o->vz += gAllz;

    if (o->vx > SPEED_LIMIT || o->vy > SPEED_LIMIT || o->vz > SPEED_LIMIT) {
      o->id = - o->id;
      //printf("%d crashed by overspeed\n", o->id);
      return;
    }
  }
}

__global__ void ThreadUpdateOrb(Orb *oList, int nOrb) {
  int i = threadIdx.x + blockIdx.x * blockDim.x;
  if (i < nOrb) {
      OrbUpdate(&oList[i], oList, nOrb);
  //} else { printf("the i exceeded:%d\n", i); // realy will be exceeded if thread/block too more than nOrb
  }
}

void PrintOrbList(Orb *oList, int nOrb) {
  for (int i=0; i<nOrb; ++i) {
      printf("[%f,%f,%f,%e,%e,%e,%f,%d]\n", oList[i].x, oList[i].y, oList[i].z, oList[i].vx, oList[i].vy, oList[i].vz, oList[i].mass, oList[i].id);
  }
}

void DiffOrbList(Orb *oList, int nOrb, Orb *oListDiff) {
  Orb oSum;
  for (int i=0; i<nOrb; ++i) {
    oSum.x += oListDiff[i].x - oList[i].x;
    oSum.y += oListDiff[i].y - oList[i].y;
    oSum.z += oListDiff[i].z - oList[i].z;
    oSum.vx += oListDiff[i].vx - oList[i].vx;
    oSum.vy += oListDiff[i].vy - oList[i].vy;
    oSum.vz += oListDiff[i].vz - oList[i].vz;
    oSum.mass += oListDiff[i].mass - oList[i].mass;
  }
  oSum.x /= double(nOrb);
  oSum.y /= double(nOrb);
  oSum.z /= double(nOrb);
  oSum.vx /= double(nOrb);
  oSum.vy /= double(nOrb);
  oSum.vz /= double(nOrb);
  oSum.mass /= double(nOrb);
  printf("avg diff:%g,%g,%g,%g,%g,%g,%g\n", oSum.x, oSum.y, oSum.z, oSum.vx, oSum.vy, oSum.vz, oSum.mass);
}

void SaveOrbList(Orb *oList, int nOrb, const char* filename) {
  FILE* f = fopen(filename, "w");
  if (f == NULL) {
    printf("Error opening file %s!\n", filename);
    return;
  }
  fputs("[", f);
  for (int i=0; i<nOrb; ++i) {
      fprintf(f, "[%.15g,%.15g,%.15g,%.15g,%.15g,%.15g,%g,%d]", oList[i].x, oList[i].y, oList[i].z, oList[i].vx, oList[i].vy, oList[i].vz, oList[i].mass, oList[i].id);
      if (i < nOrb-1) {
        fputs(",", f);
      }
  }
  fputs("]", f);
  fclose(f);
}

const char *loadFile = "";
const char *saveFile = "list1.json";

// need exclusive parameter
void* ThreadSavingOrbList(void* ptr) {
  //OrbList *oList = (OrbList*)ptr;
  SavingThreadParam *param = (SavingThreadParam*)ptr;
  while (param->state == 1) {
    usleep(500000);
    SaveOrbList(param->list, param->n, saveFile);
  }
  return NULL;
}

Orb *LoadOrbList(const char* loadFile, int *nOrbLoaded) {
    Orb *oList = NULL;
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
    oList = (Orb*)malloc(nOrb * sizeof(Orb));

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
              sscanf(restLine+lastLeftBracket+1, "%lf,%lf,%lf,%lf,%lf,%lf,%lf,%d", 
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

// ./Orbs -n 3 -t 40000 -l list1.json -s list1.json
int main(int argc, char *argv[]) {
    int nOrb = 3;
    int nTimes = 40000;
    srand(time(NULL));

    // Parse arguments
    if (argc >= 2) {
      for (int i = 0; i < argc; ++i) {
        if (strcmp(argv[i], "-n") == 0 && i+1 < argc) {
            nOrb = atoi(argv[i+1]);
        }
        if (strcmp(argv[i], "-t") == 0 && i+1 < argc) {
            nTimes = atoi(argv[i+1]);
        }
        if (strcmp(argv[i], "-l") == 0 && i+1 < argc) {
            loadFile = argv[i+1];
        }
        if (strcmp(argv[i], "-s") == 0 && i+1 < argc) {
            saveFile = argv[i+1];
        }
      }
    }

    // 申请host内存
    Orb *oList;// = (Orb*)malloc(nOrb * sizeof(Orb));
    Orb *oList2;// = (Orb*)malloc(nOrb * sizeof(Orb));

    // 初始化数据
    if (strcmp(loadFile, "") == 0) {
      oList = (Orb*)malloc(nOrb * sizeof(Orb));
      oList2 = (Orb*)malloc(nOrb * sizeof(Orb));
      for (int i = 0; i < nOrb; ++i) {
        oList[i].id = i+1;
        oList[i].mass = (double)rand() / RAND_MAX * MASS_RANGE;
        double radius = DISTRI_WIDE * (double)rand() / RAND_MAX;
        double idx = (double)rand() / RAND_MAX * PI * 2;
        oList[i].x = cos(idx) * radius;
        oList[i].y = sin(idx) * radius;
        oList[i].z = ((double)rand() / RAND_MAX - 0.5)*2*DISTRI_WIDE/1000;
        oList[i].vx = cos(idx+PI/2.0) * VELO_RANGE;
        oList[i].vy = sin(idx+PI/2.0) * VELO_RANGE;
      }
      memcpy(oList2, oList, nOrb*sizeof(Orb));
    } else {
      // load file from json
      oList = LoadOrbList(loadFile, &nOrb);
      if (oList == NULL) {
	 printf("load from loadFile %s failed\n", loadFile);
	 return 0;
      }
      oList2 = (Orb*)malloc(nOrb *sizeof(Orb));
      memcpy(oList2, oList, nOrb*sizeof(Orb));
    }
    //PrintOrbList(oList, nOrb);

    // 申请device内存
    Orb *doList;
    cudaMalloc((void**)&doList, nOrb*sizeof(Orb));

    // 将host数据拷贝到device
    cudaMemcpy((void*)doList, (void*)oList, nOrb*sizeof(Orb), cudaMemcpyHostToDevice);
    
    // 定义kernel的执行配置 // only 1024 work well
    dim3 blockSize(1024);
    dim3 gridSize((nOrb + blockSize.x - 1) / blockSize.x);

    printf("init ok, nOrb:%d nTimes:%d, will times:%ld loadFile:%s gridSize:%d blockSize:%d\n", nOrb, nTimes, long(nOrb)*long(nOrb)*long(nTimes), loadFile, gridSize.x, blockSize.x);

    // Start a thread to save orb list
    SavingThreadParam param = {oList2, nOrb, 1};
    pthread_create(&param.tid, NULL, ThreadSavingOrbList, &param);

    clock_t timeStart = clock();

    // 执行kernel
    for (int i=0; i<nTimes; ++i) {
      ThreadUpdateOrb <<< gridSize, blockSize >>>(doList, nOrb);
      cudaDeviceSynchronize(); //调用次数越少越好
      if (nTimes >= 10 && (i+1)%(nTimes/10) == 0) {
        printf("process:%d/%d, time:%.3f cps:%e estimate remain:%.3fs\n", i+1, nTimes, (double(clock()-timeStart)/CLOCKS_PER_SEC), double(long(nOrb)*long(nOrb)*long(i+1))/(double(clock()-timeStart)/CLOCKS_PER_SEC), double(nTimes-i-1)/double(i+1)*(double(clock()-timeStart)/CLOCKS_PER_SEC));
        cudaMemcpy((void*)oList2, (void*)doList, nOrb*sizeof(Orb), cudaMemcpyDeviceToHost);
      }
    }

    param.state = 0; // stop the thread
    // 将device得到的结果拷贝到host
    cudaMemcpy((void*)oList2, (void*)doList, nOrb*sizeof(Orb), cudaMemcpyDeviceToHost);

    clock_t timeEnd = clock();
    DiffOrbList(oList, nOrb, oList2);
    SaveOrbList(oList2, nOrb, saveFile);

    // 检查执行结果
    printf("all done. nOrb:%d times:%ld use time:%f cps:%e\n", 
      nOrb, 
      long(nOrb)*long(nOrb)*long(nTimes), 
      double(timeEnd-timeStart)/CLOCKS_PER_SEC, 
      double(long(nOrb)*long(nOrb)*long(nTimes))/(double(timeEnd-timeStart)/CLOCKS_PER_SEC));
    
    // 释放device内存 & 释放host内存
    cudaFree(doList);
    free(oList);
    free(oList2);

    return 0;
}
