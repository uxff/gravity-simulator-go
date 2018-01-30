/*
	本文件包含不常用函数
*/

package main

import (
	"fmt"
	"math"
	"math/rand"
)

/*思路不好，作废*/
func (this *FlowList) Move2(m *Topomap, w *WaterMap) {
	ox, oy := this.lastDot.x, this.lastDot.y

	// 流入水源
	oid := int(ox) + int(oy)*m.width
	if oid < 0 || m.width*m.height < oid {
		fmt.Println("illegal oid:", oid)
		return
	}

	// lastDot indicate next
	//var nextDot = &this.List[this.length]
	var nextDot *WaterDot = nil

	// 假设选择一个方向 查看是否能前进
	var allvx, allvy float64 = 0.0, 0.0
	var theDir, rollDir float64 = this.lastDot.dir, 0.0
	var tx, ty int

	var hasStep, needTurn bool = false, false
	for i := 0; i < 20; i++ {
		var pit1 float64 = rand.Float64() - rand.Float64()
		rollDir = pit1 * pit1 * pit1 * math.Pi / 2.0

		theDir += rollDir
		allvx, allvy = (math.Cos(theDir) * this.step), (math.Sin(theDir) * this.step)

		// 碰到边界
		tx, ty = int(float64(this.lastDot.x)+allvx), int(float64(this.lastDot.y)+allvy)
		if tx < 0 || ty < 0 || int(tx) > m.width-1 || int(ty) > m.height-1 {
			//fmt.Println("seems over bound tx,ty:", tx, ty, "o=", this.lastDot, "i=", i)
			continue
		}

		// 地形较高 不允许流向高处 对方海拔+对方水位-本地海拔+本地水位
		assumeFall := int(m.data[ty*m.width+tx]) - int(m.data[oid]) // + len(w.data[ty*m.width+tx].input) - len(w.data[oid].input)
		if assumeFall >= 1 {
			fmt.Println("seems flow up, fall,dot,tx,ty=", assumeFall, this.lastDot, tx, ty, "i=", i)
			continue
		}

		// 碰到自己 碰到水域
		if w.data[tx+ty*w.width].q >= this.lastDot.q {
			//qFall := w.data[tx+ty*w.width].q - this.lastDot.q
			//fmt.Println("seems turn self, qFall:", qFall, "i=", i)
			// 如果流入的流量小于流出的流量 则死循环
			var prepreFall int = 0
			for _, dotidx := range w.data[tx+ty*w.width].input {
				if dotidx == oid {
					//fmt.Println("do not count self:", dotidx)
					continue
				}
				prepreFall += w.data[dotidx].q
			}
			if w.data[tx+ty*w.width].q >= prepreFall+1 {
				fmt.Println("seemes turn self, too quantity", w.data[tx+ty*w.width], "oid=", oid, "i=", i, "length=", this.length)
				needTurn = true
				//continue
			}
		}

		hasStep = true
		//fmt.Println("got dir ok: theDir=", theDir, "rollDir=", rollDir, "target x,y,h:", tx, ty, assumeFall, "needTurn", needTurn)
		break
	}

	if hasStep {
		this.lastDot.nextIdx = tx + ty*w.width //int(nextDot.x) + int(nextDot.y)*m.width //nextDot
		nextDot = &w.data[tx+ty*w.width]
		nextDot.x, nextDot.y = this.lastDot.x, this.lastDot.y
		nextDot.x += float32(allvx)
		nextDot.y += float32(allvy)
		nextDot.dir = theDir
		nextDot.q++
		// 流入流量 应该排重
		nextDot.input = append(nextDot.input, oid)

		this.lastIdx = this.lastDot.nextIdx
		this.List[this.length] = this.lastDot.nextIdx

		// 切换指针
		this.lastDot = nextDot
		this.length++
		fmt.Println("go next ok:", nextDot, "rollDir=", rollDir, "needTurn=", needTurn, "length=", this.length)
	} else {
		//this.lastDot.q--
		fmt.Println("move failed, no step can go")
	}
}

// 随机洒水法：
/*
	随机在地图中选择点，并滴入一滴水，记录水位+1，尝试计算流出方向(判断旁边的水流方向)
	如果没有流出方向，水位+1；如果有流出方向，按方向滴入下一位置,本地水位-1
	向WaterMap中的某个坐标注水
	水尝试找一个流动方向
	注水，选择方向，流动耦合
    todo: 思路：多次计算，找出最合适的出口
*/
func (w *WaterMap) InjectWater3(pos int, m *Topomap) bool {
	//log.Println("START INJECT: pos=", pos)
	if pos >= len(w.data) {
		return false
	}
	var curX, curY int = pos % w.width, pos / w.height
	// 水源流向关系
	var curDot *WaterDot = &w.data[pos]

	// 流量过量则退出，否则栈溢出
	if curDot.q > 255 {
		// 理应断开与上下游关系，产生积水
		if curDot.hasNext {
			if w.data[curDot.nextIdx].q > 250 {
				log.Println("seems flow in circle, cut me->next, me=:", curDot)
				curDot.h++
				curDot.hasNext = false
				for _, itsInputIdx := range curDot.input {
					w.data[itsInputIdx].hasNext = false
				}
				curDot.input = []int{}
			} else {
				log.Println("none circle exceed occur? next q=", w.data[curDot.nextIdx].q)
			}
		}
		log.Println("too much quantity me=", curDot)
		return false
	}

	curDot.q++

	// if have output, go output
	if curDot.hasNext {
		// 查看下一个dot的input中有没有me
		var nextHasMe bool = false
		for _, itsInputIdx := range w.data[curDot.nextIdx].input {
			if itsInputIdx == pos {
				nextHasMe = true
				break
			}
		}
		// 如果没有，则将me加入到它的input
		if nextHasMe == false {
			w.data[curDot.nextIdx].input = append(w.data[curDot.nextIdx].input, pos)
		}
		//w.data[curDot.nextIdx].h++
		return w.InjectWater(curDot.nextIdx, m)
		return true
	}

	// try to find dir for flowing out
	var theDir, rollDir float64 = curDot.dir, 0.0
	var tx, ty, tIdx int
	var allvx, allvy float64 = 0.0, 0.0
	//curDot.x, curDot.y = float32(curX)+0.5, float32(curY)+0.5

	//curDot.dir = avg(curDot.input.dir)
	var hasDir bool
	curDot.dir, hasDir = curDot.calcInputAvgDir(w)
	if !hasDir {
		curDot.dir = rand.Float64() * math.Pi * 2.0
	}

	// 选择可继续流出的方向
	for i := 0; i < 20; i++ {
		// 查找较低的坑 优先流向低处
		lowest, lowestVal := curDot.getLowestNeighbor(w, m)
		if lowest < int(m.data[pos]) {
			// 低了按照此方向走
			tx, ty = lowestVal.x, lowestVal.y
			curDot.hasNext = true
			tIdx = ty*m.width + tx
			log.Println("lower first curDot,tx,ty=", curDot, tx, ty)
			break
		} else if lowest > int(m.data[pos])+curDot.h {
			curDot.hasNext = false
			curDot.dir += math.Pi
			log.Println("desire to go out, continue to try, curDot,m.h,pos=", curDot, int(m.data[pos]), pos, ",lowest,lowestVal", lowest, lowestVal)
			//break
			continue
		}

		var pit1 float64 = rand.Float64() - rand.Float64()
		rollDir = pit1 * pit1 * pit1 * (math.Pi/1.2 + float64(i)/10.0)
		//rollDir = rand.Float64() * math.Pi * 2.0
		theDir = rollDir + curDot.dir

		allvx, allvy = (math.Cos(theDir)), (math.Sin(theDir))
		// 碰到边界
		tx, ty = int(float64(curDot.x)+allvx), int(float64(curDot.y)+allvy)
		if tx < 0 || ty < 0 || int(tx) > w.width-1 || int(ty) > w.height-1 {
			log.Println("over bound tx,ty:", tx, ty, "o=", curDot, "i=", i)
			//continue
			// 任其流出地图外，不再让其流回来
			//curDot.h--
			return false
		}
		if tx == curX && ty == curY {
			// 内部移动后还在内部，将下次的基点延长
			curDot.x, curDot.y = curDot.x+float32(allvx)/2, curDot.y+float32(allvy)/2
			//log.Println("inner step")
			continue
		}
		// 计算地形落差 地形较高 不允许流向高处
		tIdx = ty*m.width + tx

		assumeFall := (int(m.data[tIdx])*1 + w.data[tIdx].h) - (int(m.data[pos])*1 + curDot.h)
		if assumeFall >= 1 {
			curDot.dir += math.Pi
			log.Println("flow up, fall,dot,tx,ty=", assumeFall, curDot, tx, ty, "i=", i)
			continue
		}

		// 对方的nextIdx不能是me
		if w.data[tIdx].hasNext && w.data[tIdx].nextIdx == pos {
			//continue
			// 撤销对方指向me的next，如果我方地形高
			if assumeFall < 0 {
				log.Println("discard target next direct me,target=", curDot, w.data[ty*m.width+tx])
				w.data[tIdx].hasNext = false
			} else {
				log.Println("cannot flow to target because its next is me: me=", curDot, "target=", w.data[ty*m.width+tx], "i=", i)
				continue
			}
		}

		// 不能交叉 如果target是斜对面，相邻的不能是next关系
		crossM := (tx - curX) * (ty - curY)
		if crossM == 1 || crossM == -1 {
			// 流向斜对面
			near1x, near1y := tx, curY
			near2x, near2y := curX, ty
			if w.data[near1x+near1y*w.width].hasNext && w.data[near1x+near1y*w.width].nextIdx == (near2x+near2y*w.width) {
				// 下游是2
				log.Println("become cross:x,y,tx,ty=", curX, curY, tx, ty, "redir to x,y=", near2x, near2y)
				tx, ty = near2x, near2y
			} else if w.data[near2x+near2y*w.width].hasNext && w.data[near2x+near2y*w.width].nextIdx == (near1x+near1y*w.width) {
				// 下游是1
				log.Println("become cross:x,y,tx,ty=", curX, curY, tx, ty, "redir to x,y=", near1x, near1y)
				tx, ty = near1x, near1y
			}
			tIdx = ty*m.width + tx
		}

		curDot.hasNext = true
		break
	}

	// 选择成功则流入
	if curDot.hasNext {
		log.Println("got dir ok: rollDir=", rollDir, "pos xy=", pos, curX, curY, "target x,y:", tx, ty, "curH,tarH=", m.data[pos], m.data[tIdx])
		// 不能过于激进地修改方向，否则曲线太弯曲容易eatself
		//		if curDot.dir-theDir < -math.Pi {
		//			curDot.dir = (curDot.dir+theDir)/2.0 - math.Pi
		//		} else if curDot.dir-theDir > math.Pi {
		//			curDot.dir = -(curDot.dir+theDir)/2.0 - math.Pi
		//		} else {
		//			curDot.dir = curDot.dir + (curDot.dir-theDir)/2.0 //(curDot.dir + theDir) / 2.0
		//		}
		//curDot.dir = theDir
		curDot.dir = curDot.dir + rollDir/2.0

		curDot.nextIdx = tIdx //tx + w.width*ty
		// 查看下一个dot的input中有没有me
		var nextHasMe bool = false
		for _, itsInputIdx := range w.data[curDot.nextIdx].input {
			if itsInputIdx == pos {
				nextHasMe = true
				break
			}
		}
		// 如果没有，则将me加入到它的input
		if nextHasMe == false {
			w.data[curDot.nextIdx].input = append(w.data[curDot.nextIdx].input, pos)
			w.data[curDot.nextIdx].giveInputXY(curDot.x+float32(allvx), curDot.y+float32(allvy))
		}
		//w.data[curDot.nextIdx].h++
		return w.InjectWater(curDot.nextIdx, m)
	} else {
		log.Println("cannot flow anywhere: curDot=", curDot)
		// 积水
		curDot.h++
		// 切断curDot跟下游的关系 本来就是断的
		//curDot.hasNext = false
		// 切断curDot跟上游的关系
		curDot.input = make([]int, 0)
		for _, itsInputIdx := range curDot.input {
			w.data[itsInputIdx].hasNext = false
		}
		// 反流 往上反馈此路不通 让上路重新选择
		for _, itsInputIdx := range curDot.input {
			w.InjectWater(itsInputIdx, m)
		}
	}
	return false
}
