package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"

	//"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/uxff/gravity-simulator-go/go_server/orbs"
	"github.com/uxff/gravity-simulator-go/go_server/saver"
)

// MaxParticles 默认最大天体数量
const MaxParticles = 100

// LoopTimes 默认计算步数
const LoopTimes = 10000

var theSaver = saver.Saver{}

func main() {
	numOrbs := MaxParticles
	numTimes := LoopTimes
	var numCpu int

	flag.IntVar(&numOrbs, "init-orbs", 0, "how many orbs init, do init when its value >1")
	flag.IntVar(&numTimes, "calc-times", 100, "how many times calc")
	var doShowList = flag.Bool("showlist", false, "show orb list and exit")
	var configMass = flag.Float64("config-mass", 10.0, "init mass of each single orb, approximately")
	var configWide = flag.Float64("config-wide", 1000.0, "init wide of orbs distrubuted")
	var configVelo = flag.Float64("config-velo", 0.005, "init velo of each single orb, approximately")
	var configStyleArrange = flag.Int("config-arrange", 3, "init style of orbs arrangement: 0=line,1=cube,2=disc,3=sphere")
	var configStyleAssemble = flag.Int("config-assemble", 2, "init style of orbs aggregation: 0=avg,1=ladder,2=variance,3=4th power")
	var bigMass = flag.Float64("bigmass", 15000.0, "config of big mass orb, like a blackhole in center, 0 means no bigger")
	var bigNum = flag.Int("bignum", 1, "config of number of big mass orbs, generally center has 1")
	var bigMassStyle = flag.Int("bigstyle", 0, "config of big mass orb distribute style: 0=center,1=outer edge,2=middle of a radius,3=random")
	var configCpu = flag.Int("config-cpu", 0, "how many cpu u want use, 0=all")
	var savePath = flag.String("savepath", "file://./", "where to save, support mc/file/redis\n\tlike: file://./filecache/")
	var saveKey = flag.String("savekey", "thelist1", "key name to save, like key of memcache, or filename in save dir")
	var loadPath = flag.String("loadpath", "", "where to load, support mc/file/redis\n\tlike: file://./filecache/, use savepath if no given")
	var loadKey = flag.String("loadkey", "", "key name to load, like key of memcache, or filename in save dir, use savekey if no given")
	var doMerge = flag.Bool("domerge", false, "merge from loadkey to savekey if true, replace if false")
	var moveExp = flag.String("moveexp", "", "move expression, like: x=-150&vy=+0.01&m=+20 only position,velo,mass")
	var saveDuration = flag.Int("save-duration", 500, "save to savepath per millisecond, 100 means 100ms")

	// flags 读取参数，必须要调用 flag.Parse()
	flag.Parse()
	log.SetFlags(0)

	if *configCpu > 0 {
		numCpu = *configCpu
	} else {
		numCpu = runtime.NumCPU() - 1
	}
	runtime.GOMAXPROCS(numCpu)

	// 如果没有设置loadkey，则使用savekey
	if len(*loadKey) == 0 {
		loadKey = saveKey
	}
	// 如果没有设置loadpath，则使用savepath
	if len(*loadPath) == 0 {
		loadPath = savePath
	}

	theSaver.SetLoadpath(loadPath)
	theSaver.SetSavepath(savePath)

	var oList []orbs.Orb

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	// 如果配置了 -init-orbs 100 参数，则不会使用loadkey
	if numOrbs > 0 {
		initConfig := orbs.InitConfig{*configMass, *configWide, *configVelo, *configStyleArrange, *configStyleAssemble, *bigMass, *bigNum, *bigMassStyle}
		oList = orbs.InitOrbs(numOrbs, &initConfig)
	} else {
		oList = theSaver.GetList(loadKey)
		// 合并 取出savekey的数据，合并loadkey的数据后存放到savekey
		if *doMerge {
			if *loadKey == *saveKey {
				fmt.Println("loadkey must not equal to save key when merge")
			} else {
				mList := theSaver.GetList(saveKey)
				// oList = append(oList, mList...)
				// // 重置id
				// for i := 0; i < len(oList); i++ {
				// 	if oList[i].Id < 0 {
				// 		oList[i].Id = -int32(i + 1)
				// 	} else {
				// 		oList[i].Id = int32(i)
				// 	}
				// }
				oMaxId := int32(0)
				for i := 0; i < len(oList); i++ {
					if oList[i].Id > oMaxId {
						oMaxId = oList[i].Id
					}
				}
				for i := 0; i < len(mList); i++ {
					mList[i].Id = oMaxId + 1 + int32(i)
					oList = append(oList, mList[i])
				}
				theSaver.SaveList(saveKey, oList)
			}
		}
		numOrbs = len(oList)
	}
	if *doShowList {
		fmt.Println(oList)
		return
	}

	// 执行批量操作，整体数据修改 比如改变位置 改变速度 改变质量
	if len(*moveExp) > 0 {
		expQuery := strings.Split(*moveExp, "&")
		expParamMap := make(map[string]string)
		for i := range expQuery {
			sTmp := strings.Split(expQuery[i], "=")
			if len(sTmp) > 1 {
				// 除法在命令行应该写作:  -moveexp "x=\/0.4" 不然会转义成git安装目录
				// 处理多余的转义符 蛋疼的gitbash
				if sTmp[1][0] == '\\' {
					sTmp[1] = sTmp[1][1:]
				}
				expParamMap[sTmp[0]] = sTmp[1]
			}
		}

		fmt.Println("moveExp=", *moveExp)

		getOperVal := func(operChar byte, leftVal, paramVal float64) (retVal float64) {
			switch operChar {
			case '*':
				leftVal *= paramVal
			case '/':
				leftVal /= paramVal
			case '-':
				leftVal -= paramVal
			case '+':
				leftVal += paramVal
			default:
			}
			return leftVal
		}

		for i := 0; i < len(oList); i++ {
			o := &oList[i]
			for ek := range expParamMap {
				s := expParamMap[ek]
				if len(s) < 2 {
					fmt.Println("illegal move exp:", s)
					continue
				}
				vTmp, _ := strconv.ParseFloat(s[1:], 64)
				switch ek {
				// x,y,x 属于坐标移动 +-*/ 支持四种运算
				case "x":
					o.X = getOperVal(s[0], o.X, vTmp)
				case "y":
					o.Y = getOperVal(s[0], o.Y, vTmp)
				case "z":
					o.Z = getOperVal(s[0], o.Z, vTmp)
				case "vx":
					o.Vx = getOperVal(s[0], o.Vx, vTmp)
				case "vy":
					o.Vy = getOperVal(s[0], o.Vy, vTmp)
				case "vz":
					o.Vz = getOperVal(s[0], o.Vz, vTmp)
				case "m":
					o.Mass = getOperVal(s[0], o.Mass, vTmp)
				}
			}
		}

	}

	fmt.Printf("start calc, orbs:%d will times:%d use cpu core:%d allMass=%e\n", numOrbs, int64(numTimes)*int64(numOrbs)*int64(numOrbs), numCpu, orbs.GetAllMass(oList))

	//realTimes, perTimes, tmpTimes := 0, 0, 0
	startTimeNano := time.Now().UnixNano()

	go func() {
		tmpTimes := 0
		for {
			time.Sleep(time.Millisecond * time.Duration(*saveDuration))
			theSaver.SaveList(saveKey, oList)
			tmpTimes++
			if tmpTimes > 10 {
				tmpTimes = 0
				oList = orbs.ClearOrbList(oList)
			}
		}
	}()

	realTimes := orbs.UpdateOrbs(oList, numTimes)
	//	for i := 0; i < numTimes; i++ {
	//		perTimes = orbs.UpdateOrbs(oList, i)
	//		realTimes += perTimes
	//		//		tmpTimes += perTimes
	//		//		if tmpTimes > 10000000 {
	//		//			theSaver.SaveList(saveKey, oList)
	//		//			if i%10 == 1 { //orbs.GetCrashed()%10 == 9 &&
	//		//				oList = orbs.ClearOrbList(oList)
	//		//			}
	//		//			tmpTimes = 0
	//		//		}
	//	}

	oList = orbs.ClearOrbList(oList)
	//fmt.Println("when clear oList=", oList)

	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Printf("after calc, orbs:%d real times:%d used time:%6fs CPS:%e\n", len(oList), realTimes, timeUsed, float64(realTimes)/timeUsed)
	orbs.ShowMonitorInfo(oList)

	theSaver.SaveList(saveKey, oList)

	endTimeNano = time.Now().UnixNano()
	timeUsed2 := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Printf("all used time with save:%6fs saveTimes:%d save/sec:%.2f clearTimes:%d crashed:%d\n", timeUsed2, theSaver.GetSavetimes(), float64(theSaver.GetSavetimes())/timeUsed, orbs.GetClearTimes(), orbs.GetCrashed())

	theSaver.Clear()
}

/*
	todo list:
	实现多服务器计算
*/
