/*
天体及天体计算框架
*/
package orbs

import (
	"fmt"
	"log"
	"math"
)

// 天体结构体声明
type Orb struct {
	X    float64 `json:"x"`  // 坐标x
	Y    float64 `json:"y"`  // 坐标y
	Z    float64 `json:"z"`  // 坐标z
	Vx   float64 `json:"vx"` // 速度x
	Vy   float64 `json:"vy"` // 速度y
	Vz   float64 `json:"vz"` // 速度z
	Mass float64 `json:"m"`  // 质量
	Id   int32   `json:"id"` // Id<0表示状态不正常 不能参与计算,不能当下标使用,只能参与比较
	//Stat int32   `json:"st"` // 用于标记是否已爆炸 1=正常 2=已爆炸 //作废 Id instead
	//Size int     `json:"sz"` // 大小，用于计算吞并的天体数量 //作废 Mass instead
	//idx       int
	//crashedBy int
}

// 加速度结构体
type Acc struct {
	Ax float64
	Ay float64
	Az float64
	A  float64
}

// 碰撞事件
type CrashEvent struct {
	Idx       int
	CrashedBy int
	Reason    int8
}

// 万有引力常数
const G = 0.000005

// 最小天体距离值 两天体距离小于此值了会相撞 应当远大于速度 比如大于速度1000倍以上 如果考虑斥力则使用小于1的值比较合适
const MIN_CRITICAL_DIST = 0.2

// 天体速度差大于此值时，会被撕裂 问题: 质量凭空丢失?
const SPEED_LIMIT = 3.0

// 监控速度和加速度
var maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMass, allMass float64 = 0, 0, 0, 0, 0, 0, 0, 0
var maxMassId int32 = 0
var clearTimes, willTimes, realTimes int64 = 0, 0, 0

var c chan int                     //= make(chan int, 10000)	// orb.update()完成队列
var crashEventChan chan CrashEvent //= make(chan CrashEvent, 0) // 撞击事件队列

var nCount, nCrashed int = 0, 0

func UpdateOrbs(oList []Orb, numTimes int) int64 {
	realTimes = 0
	//theListLength = len(oList)
	willTimes = int64(len(oList)) * int64(len(oList)) * int64(numTimes)
	// 初始化chan CrashEvent ,orb.update()将会往crashEventChan中push事件
	// 事件队列，提升效率 15%左右
	crashEventChan = make(chan CrashEvent, len(oList))
	// 分配足够的队列空间，提升效率 0.5%左右
	c = make(chan int, len(oList))

	for i := 0; i < numTimes; i++ {
		realTimes += UpdateOrbsOnce(oList, i)
	}
	return realTimes
}

// 所有天体运动一次
func UpdateOrbsOnce(oList []Orb, nStep int) int64 {
	thelen := len(oList)
	nCount := 0
	var o, target *Orb
	// var targetMassOld float64
	for i := 0; i < thelen; i++ {
		//oList[i].idx = i
		//oList[i].crashedBy = -1
		go func(i int) { oList[i].Update(oList, i) }(i)
	}
	for {
		if nCount >= thelen {
			break
		}

		select {
		case <-c:
			// 正常计算完成任务返回
			nCount++
		case anEvent := <-crashEventChan:
			nCrashed++
			// 收集事件队列信息
			o = &oList[anEvent.Idx]
			// 只处理自己被谁撞击合并
			target = &oList[anEvent.CrashedBy]
			log.Println("a CrashEvent:", o.Id, "crashed by", target.Id, "index:", anEvent, "nCrashed:", nCrashed, "nStep:", nStep)
			// targetMassOld = target.Mass
			target.Mass += o.Mass
			// 碰撞后动量传递 有必要？
			// target.Vx = (targetMassOld*target.Vx + o.Mass*o.Vx) / target.Mass
			// target.Vy = (targetMassOld*target.Vy + o.Mass*o.Vy) / target.Mass
			// target.Vz = (targetMassOld*target.Vz + o.Mass*o.Vz) / target.Mass
			o.Mass = 0
			//o.Stat = 2
		}
	}
	return int64(thelen) * int64(nCount)
}

// 天体运动一次
func (o *Orb) Update(oList []Orb, idx int) {
	// 先把位置移动起来，再计算环境中的加速度，再更新速度，为了更好地解决并行计算数据同步问题
	if o.Id > 0 /*o.Stat == 1*/ {
		aAll := o.CalcGravityAll(oList, idx)
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		// 监控速度和加速度
		// isMeRipped := dist < math.Sqrt(o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz)*8
		if isMeRipped := o.Vx > SPEED_LIMIT || o.Vy > SPEED_LIMIT || o.Vz > SPEED_LIMIT; isMeRipped {
			// crashReason = crashReason | 2
			o.Id = -o.Id //o.Stat = 2 // 此处必须对自己标记，否则会出现被多个ta撞击的事件
			crashEventChan <- CrashEvent{idx, 0, 2}
		}
		if maxVeloX < math.Abs(o.Vx) {
			maxVeloX = o.Vx
		}
		if maxVeloY < math.Abs(o.Vy) {
			maxVeloY = o.Vy
		}
		if maxVeloZ < math.Abs(o.Vz) {
			maxVeloZ = o.Vz
		}
		if maxAccX < math.Abs(aAll.Ax) {
			maxAccX = aAll.Ax
		}
		if maxAccY < math.Abs(aAll.Ay) {
			maxAccY = aAll.Ay
		}
		if maxAccZ < math.Abs(aAll.Az) {
			maxAccZ = aAll.Az
		}
		if maxMass < o.Mass {
			maxMass = o.Mass
			maxMassId = o.Id
		}
	}
	c <- idx //len(oList)
}

// 计算天体受到的总体引力
func (o *Orb) CalcGravityAll(oList []Orb, idx int) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if /*target.Stat != 1 || o.Stat != 1*/ target.Id < 0 || o.Id < 0 || target.Id == o.Id {
			continue
		}

		dist := o.CalcDist(target)

		crashReason := int8(0)
		// 距离太近，被撞
		if isTooNearly := dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST; isTooNearly {
			crashReason = crashReason | 1
		}
		// 速度太快，被撕裂 me ripped by ta
		// isMeRipped := dist < math.Sqrt(o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz)*8
		// if isMeRipped := o.Vx > SPEED_LIMIT || o.Vy > SPEED_LIMIT || o.Vz > SPEED_LIMIT; isMeRipped {
		// 	crashReason = crashReason | 2
		// }

		if crashReason > 0 {
			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass < target.Mass {
				// 碰撞事件交给主goroutine处理对方的质量改变，这里发送信息，不做修改操作
				o.Id = -o.Id //o.Stat = 2 // 此处必须对自己标记，否则会出现被多个ta撞击的事件
				crashEventChan <- CrashEvent{idx, i, crashReason}
				//o.crashedBy = i // 不能取target.idx // 待思考为什么 协程间数据共享，不安全
				// 由于并发数据分离，当前goroutine只允许操作当前orb,不允许操作别的orb，所以不允许操作ta的数据
				break
			}
			// no else 在循环时可能有多个o crashed ta,但是只有一个o crashed by ta
		} else {
			// 作用正常，累计计算受到的所有的万有引力
			gTmp := o.CalcGravity(&oList[i], dist)
			// ---------- 计算斥力start ----------
			// rTmp := o.CalcRepulsionF(&oList[i], dist)
			// gTmp.add(&rTmp)
			// ---------- 计算斥力end ----------
			// ---------- 附加计算旋转力start ----------
			// spinForce := o.CalcSpinningF(&oList[i], dist)
			// gTmp.add(&spinForce)
			// ---------- 附加计算旋转力end ----------
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
			gAll.Az += gTmp.Az
		}
	}

	return gAll
}

// 计算天体与目标的引力
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	// 万有引力公式
	a.A = target.Mass / (dist * dist) * G
	a.Ax = -a.A * (o.X - target.X) / dist //a.A * math.Cos(a.Dir)
	a.Ay = -a.A * (o.Y - target.Y) / dist //a.A * math.Sin(a.Dir)
	a.Az = -a.A * (o.Z - target.Z) / dist //a.A * math.Sin(a.Dir)
	return a
}

// 计算距离
func (o *Orb) CalcDist(target *Orb) float64 {
	return math.Sqrt((o.X-target.X)*(o.X-target.X) + (o.Y-target.Y)*(o.Y-target.Y) + (o.Z-target.Z)*(o.Z-target.Z))
}

func (o *Orb) MarshalJSON() (str []byte, err error) {
	strs := fmt.Sprintf("[%g,%g,%g,%g,%g,%g,%g,%d]", o.X, o.Y, o.Z, o.Vx, o.Vy, o.Vz, o.Mass, o.Id)
	return []byte(strs), nil
}
func (o *Orb) UnmarshalJSON(input []byte) error {
	_, err := fmt.Sscanf(string(input), "[%f,%f,%f,%f,%f,%f,%f,%d]", &o.X, &o.Y, &o.Z, &o.Vx, &o.Vy, &o.Vz, &o.Mass, &o.Id)
	//log.Println("when unmarshal(", string(input), ") n,err,o=", n, err, o)
	return err
}

// 设置撞击 作废
/*
func (o *Orb) SetCrashedBy(crashedBy int) {
	o.crashedBy = crashedBy
}
*/
// 清理orbList中的垃圾
func ClearOrbList(oList []Orb) []Orb {
	allMass = 0
	//var alive int = len(oList)
	for i := 0; i < len(oList); i++ {
		allMass += oList[i].Mass
		if oList[i].Id < 0 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
			//alive--
			//} else {
		}
	}
	//log.Println("when clear alive=", alive)
	clearTimes++
	return oList
}

func ShowMonitorInfo(oList []Orb) {
	log.Printf("maxVelo=%.6g %.6g %.6g maxAcc=%.6g %.6g %.6g maxMass(%d)=%e allMass=%e\n", maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMassId, maxMass, GetAllMass(oList))
}
func GetClearTimes() int64 {
	return clearTimes
}
func GetCrashed() int {
	return nCrashed
}
func GetAllMass(oList []Orb) float64 {
	allMass = 0
	for i := 0; i < len(oList); i++ {
		allMass += oList[i].Mass
	}
	return allMass
}

// func GetRealTimes() int64 {
// 	return realTimes
// }
// func GetWillTimes() int64 {
// 	return willTimes
// }
