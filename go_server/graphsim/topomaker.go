/*
	usage: ./topomaker -w 200 -h 200 -hill 10 -hill-wide 20 -flow 100 -zoom 5
    todo: table lize with http server
*/
package main

import (
	"fmt"
	"image"
	"image/color"
	"net/http"
	//"image/draw"
	"flag"
	//"image/jpeg"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	drawer "./drawer"
)

// 使用水滴滚动
type Droplet struct {
	x float32
	y float32
}

// 将变成固定不移动
type WaterDot struct {
	x        float32 // 将不变化 =Topomap[x,y] +(0.5, 0.5)
	y        float32
	xPower   float32 // v2
	yPower   float32 // v2
	dir      float64
	h        int   // 积水高度，产生积水不参与流动，流动停止 // v2将由水滴实体代替该变量
	q        int   // 流量 0=无 历史流量    //
	input    []int // 流入坐标    // v2将取消该变量
	nextIdx  int
	hasNext  bool    // 有下一个方向
	dirPower float32 // v2
}
type FlowList struct {
	List    []int
	lastIdx int
	lastDot *WaterDot
	length  int32
	step    float64
}
type Topomap struct {
	data   []uint8 // 对应坐标只保存高度
	width  int
	height int
}
type WaterMap struct {
	data   []WaterDot
	width  int
	height int
}

func (this *FlowList) Init(x int, y int, w *WaterMap, maxlen int) {
	this.List = make([]int, maxlen)
	//this.lastDot = &this.List[0]
	if maxlen > 0 {
		this.lastIdx = x + y*w.width
		this.List[0] = this.lastIdx
		this.lastDot = &w.data[this.lastIdx]
		this.lastDot.x = float32(x) + 0.5
		this.lastDot.y = float32(y) + 0.5
		this.lastDot.q = 1
		this.length = int32(maxlen)
		theDir := rand.Float64() * math.Pi * 2.0
		this.lastDot.dir = theDir
	}
	this.step = 1
	this.length = int32(maxlen)
	//log.Println("new dir:", theDir, this.lastDot)

	//w.data[x+y*w.width] = *this.lastDot
}

// 先处理场向量
// 再注水流动

// 预先处理每个点的场向量
// 假设每个点都有一个场，计算出这个场的方向
// Topomap is basic topomap
// WaterMap is empty fields of all, to be inited
// @param int ring 表示计算到几环 默认2环
func (w *WaterMap) AssignVector(m *Topomap, ring int) {
	for idx, curDot := range w.data {
		// xPoser, yPower 单位为1
		var xPower, yPower int
		// 2nd ring
		_, lowestPos := curDot.getLowestNeighbors(curDot.getNeighbors(w), m)
		for _, neiPos := range lowestPos {
			xPower += 4 * (neiPos.x - int(idx%w.width))
			yPower += 4 * (neiPos.y - int(idx/w.width))
		}

		// 3rd ring. done 三环的影响力是二环的1/4
		if ring >= 3 {
			_, lowestPos = curDot.getLowestNeighbors(curDot.get3rdNeighbors(w), m)
			for _, neiPos := range lowestPos {
				xPower += neiPos.x - int(idx%w.width)
				yPower += neiPos.y - int(idx/w.width)
			}
		}

		// 四环 四环影响力是二环的1/16
		// if ring >= 4 {
		//}

		if xPower != 0 || yPower != 0 {
			w.data[idx].dirPower = w.data[idx].dirPower + 1.0 //应该是邻居落差
			w.data[idx].dir = math.Atan2(float64(yPower), float64(xPower))
			w.data[idx].xPower, w.data[idx].yPower = float32(math.Cos(w.data[idx].dir)), float32(math.Sin(w.data[idx].dir))
		}

	}
}

func UpdateTopo() {

}
func UpdateDroplet() {

}

// v2 将使用新洒水法
// 随机洒水法：
/*
	随机在地图中选择点，并滴入一滴水，尝试计算流出方向(判断旁边的水流方向)
	如果没有流出方向，水位+1；如果有流出方向，按方向滴入下一位置,本地水位-1
	向WaterMap中的某个坐标注水
	水尝试找一个流动方向 随机roll出一个方向
	注水，选择方向，流动耦合
    todo: 思路：多次计算，找出最合适的出口
*/
func (w *WaterMap) InjectWater(pos int, m *Topomap) bool {
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
		lowest, lowestVal := curDot.getLowestNeighbor(curDot.getNeighbors(w), m)
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

func (this *Topomap) Init(width int, height int) {
	this.data = make([]uint8, width*height)
	this.width = width
	this.height = height
}
func (this *WaterMap) Init(width int, height int) {
	this.data = make([]WaterDot, width*height)
	this.width = width
	this.height = height
	// 把点的实际基点摆在中间
	for x := 0; x < this.width; x++ {
		for y := 0; y < this.height; y++ {
			this.data[x+y*this.width].x = float32(x) + 0.5
			this.data[x+y*this.width].y = float32(y) + 0.5
			this.data[x+y*this.width].hasNext = false
		}
	}
}

type Ring struct {
	x       int
	y       int
	r       int
	h       int
	tiltDir float64 // 倾斜方向
	tiltLen int     // 倾斜长度
}

/*返回颜色数组，下标越大颜色海拔越高*/
func colorTpl(colorTplFile string) []color.Color {
	var colorTplFileIo, _ = os.Open(colorTplFile)
	defer colorTplFileIo.Close()
	var colorTplPng, err = png.Decode(colorTplFileIo)

	if err != nil {
		log.Println("png.decode err when read colorTpl:", err)
		return nil
	}
	theLen := colorTplPng.Bounds().Dy()
	cs := make([]color.Color, theLen)
	for i := 0; i < theLen; i++ {
		cs[i] = colorTplPng.At(0, theLen-i-1)
	}
	return cs
}

func lineTo(img *image.RGBA, startX, startY, destX, destY int, lineColor, startColor color.Color, scale float64) {
	distM := math.Sqrt(float64((startX-destX)*(startX-destX) + (startY-destY)*(startY-destY)))
	var i float64
	//scale = 1.0
	for i = 0; i < distM*scale; i++ {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), lineColor)
	}
	// 线段最后一点 绘制成始发地地形的颜色 startColor
	if startX != destX && startY != destY {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), startColor)
	}
}

// 计算来源的平均夹角
func (d *WaterDot) calcInputAvgDir(w *WaterMap) (dir float64, hasDir bool) {
	for i, inputIdx := range d.input {
		inputDot := &w.data[inputIdx]
		if i == 0 {
			dir = inputDot.dir
			hasDir = true
		} else {
			// 合并方向 夹角大于180 取-平均值
			if dir-inputDot.dir < -math.Pi || dir-inputDot.dir > math.Pi {
				dir = -(dir + inputDot.dir) / 2.0
			} else {
				dir = (dir + inputDot.dir) / 2.0
			}
		}
	}
	return dir, hasDir
}

func (d *WaterDot) giveInputXY(inputX, inputY float32) {
	//d.x, d.y = (d.x+inputX)/2, (d.y+inputY)/2
	d.x, d.y = inputX, inputY
}

// 此函数固定返回本坐标周边2环8个边界点，可能包含超出地图边界的点
func (d *WaterDot) getNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 8)
	pos[0].x, pos[0].y = int(d.x+1), int(d.y)
	pos[1].x, pos[1].y = int(d.x+1), int(d.y-1)
	pos[2].x, pos[2].y = int(d.x), int(d.y-1)
	pos[3].x, pos[3].y = int(d.x-1), int(d.y-1)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y)
	pos[5].x, pos[5].y = int(d.x-1), int(d.y+1)
	pos[6].x, pos[6].y = int(d.x), int(d.y+1)
	pos[7].x, pos[7].y = int(d.x+1), int(d.y+1)
	return pos
}

// 此函数固定返回本点周边3环12个边界点，可能包含超出地图边界的点
func (d *WaterDot) get3rdNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 12)
	// right
	pos[0].x, pos[0].y = int(d.x+2), int(d.y)
	pos[1].x, pos[1].y = int(d.x+2), int(d.y-1)
	// top
	pos[2].x, pos[2].y = int(d.x+1), int(d.y-2)
	pos[3].x, pos[3].y = int(d.x), int(d.y-2)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y-2)
	// left
	pos[5].x, pos[5].y = int(d.x-2), int(d.y-1)
	pos[6].x, pos[6].y = int(d.x-2), int(d.y)
	pos[7].x, pos[7].y = int(d.x-2), int(d.y+1)
	// bottom
	pos[8].x, pos[8].y = int(d.x-1), int(d.y+2)
	pos[9].x, pos[9].y = int(d.x), int(d.y+2)
	pos[10].x, pos[10].y = int(d.x+1), int(d.y+2)
	// right
	pos[11].x, pos[11].y = int(d.x+2), int(d.y+1)
	return pos
}

/*获取周围最低的点 最低点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getLowestNeighbor(arrNei []struct{ x, y int }, m *Topomap) (lowestLevel int, lowestVal struct{ x, y int }) {
	//	arrNei := d.getNeighbors(w)
	//log.Println("d,arrNei=", d, arrNei)
	// 原理： highMap[high] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > m.width-1 || nei.y < 0 || nei.y > m.height-1 {
			// 超出地图边界的点
			continue
		}
		high := int(m.data[nei.x+nei.y*m.width]) + d.h
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	lowestLevel = 1000
	for k, _ := range highMap {
		if k < lowestLevel {
			lowestLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[lowestLevel]) == 0 {
		return lowestLevel, lowestVal
	}
	return lowestLevel, highMap[lowestLevel][rand.Int()%len(highMap[lowestLevel])]
}

/*获取周围最低的点 最低点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getLowestNeighbors(arrNei []struct{ x, y int }, m *Topomap) (lowestLevel int, lowestPos []struct{ x, y int }) {
	//arrNei := d.getNeighbors(w)
	//log.Println("d,arrNei=", d, arrNei)
	// 原理： highMap[high] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > m.width-1 || nei.y < 0 || nei.y > m.height-1 {
			// 超出地图边界的点
			continue
		}
		high := int(m.data[nei.x+nei.y*m.width]) + d.h
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	lowestLevel = 1000
	for k, _ := range highMap {
		if k < lowestLevel {
			lowestLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[lowestLevel]) == 0 {
		return lowestLevel, nil
	}
	return lowestLevel, highMap[lowestLevel]
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	//var times = flag.Int("flow", 5, "flow move times")
	var width, height int = 500, 500
	flag.IntVar(&width, "w", width, "width of map")
	flag.IntVar(&height, "h", width, "height of map")
	var outname = flag.String("out", "testmap", "image filename of output")
	var outdir = flag.String("outdir", "output", "out put dir")
	var nHills = flag.Int("hill", 100, "hill number for making rand topo by hill")
	var hillWide = flag.Int("hill-wide", 100, "hill wide for making rand topo by hill")
	var bShowMap = flag.Bool("print", false, "print map for debug")
	var nRidge = flag.Int("ridge", 100, "ridge times for making ridge")
	var ridgeWide = flag.Int("ridge-wide", 50, "ridge wide for making ridge")
	var ridgeStep = flag.Int("ridge-step", 8, "ridge step for making ridge")
	var zoom = flag.Int("zoom", 1, "zoom of out put")
	var riverArrowScale = flag.Float64("river-arrow-scale", 0.8, "river arrow scale")

	flag.Parse()

	var m Topomap
	var w WaterMap
	var ridge FlowList
	//var river FlowList

	// 初始化 watermap topomap
	w.Init(width, height)
	m.Init(width, height)

	// 初始化 非流动
	//river.Init(width/2, height/2, &w, *times+1)
	ridge.Init(width/2, height/2, &w, *nRidge)
	ridge.step = float64(*ridgeStep)

	if _, derr := os.Open(*outdir); derr != nil {
		log.Println("output dir seems not exist:", *outdir, derr)
		if cerr := os.Mkdir(*outdir, os.ModePerm); cerr != nil {
			log.Println("os.mkdir:", *outdir, cerr)
			//return
		}
	}

	//picFile, _ := os.Create(*outname + ".jpg")
	filename := fmt.Sprintf("%s-%s", *outname, time.Now().Format("20060102150405"))
	picFile2, _ := os.Create(*outdir + "/" + filename + ".png")
	defer picFile2.Close()

	// 随机n个圆圈 累加抬高
	rings := make([]Ring, *nHills)
	for ri, _ := range rings {
		r := &rings[ri]
		r.x, r.y, r.r, r.h = (rand.Int() % width), (rand.Int() % height), (rand.Int()%(*hillWide) + 1), (rand.Int()%(5) + 2)
		r.tiltDir, r.tiltLen = rand.Float64()*math.Pi, (rand.Int()%10)+1
	}

	// 转换痕迹为ridge 为每个环分配随机半径
	ridgeRings := make([]Ring, ridge.length)
	baseTowardX, baseTowardY := (rand.Int()%width-width/2)/20, (rand.Int()%height-height/2)/20
	for ri := 0; ri < int(ridge.length); ri++ {
		r := &ridgeRings[ri]
		if ri == 0 {
			// 第一个
			//r.x, r.y, r.r = ridge.List[ri]%width, ridge.List[ri]/width, (rand.Int() % (*ridgeWide))
			r.x, r.y, r.r, r.h = (rand.Int() % width), (rand.Int() % height), (rand.Int()%(*ridgeWide) + 1), (rand.Int()%(5) + 2)
		} else {
			// 其他
			r.x, r.y, r.r, r.h = ridgeRings[ri-1].x+(rand.Int()%*ridgeWide)-*ridgeWide/2+baseTowardX, ridgeRings[ri-1].y+(rand.Int()%*ridgeWide)-*ridgeWide/2+baseTowardY, (rand.Int()%(*ridgeWide) + 1), (rand.Int()%(5) + 2)
		}
	}
	log.Println("ridgeRings=", ridgeRings)

	// 生成地图 制造地形
	var tmpColor, maxColor float32 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = 0
			// 收集ridgeRings产生的attitude
			for _, r := range ridgeRings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				rn := (r.r)
				if distM <= r.r*r.r {
					//tmpColor++
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(distM) / float32(r.r*r.r) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}
			// 收集rings产生的attitude
			for _, r := range rings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				//rn := float64(r.tiltLen)*math.Sin(r.tiltDir-math.Atan2(float64(y), float64(x))) + float64(r.r)	// 尝试倾斜地图中的圆环 尝试失败
				rn := (r.r)
				if distM <= int(rn*rn) {
					// 产生的ring中间隆起
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(distM) / float32(rn*rn) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor, "r=", r)
				}
			}

			m.data[x+y*width] = uint8(tmpColor) //+ uint8(rand.Int()%2) //int8(width - x)
		}
	}
	maxColor *= 2

	w.AssignVector(&m, 3)

	// 随机洒水
	//	for i := 0; i < *times; i++ {
	//		idx := rand.Int() % len(w.data)
	//		//x, y := idx%w.width, idx/w.width
	//		//w.data[idx].x, w.data[idx].y = float32(x)+0.5, float32(y)+0.5
	//		w.data[idx].hasNext = false
	//		w.InjectWater(idx, &m)
	//	}

	img := image.NewRGBA(image.Rect(0, 0, width**zoom, height**zoom))

	DrawToImg(img, &m, &w, maxColor, zoom, riverArrowScale)

	go drawer.StartHtmlDrawer(":33399")
	DrawToHtml(&w, &m)
	log.Printf("drow to html ok, open localhost:33339 and view")

	// 输出图片文件
	go func() {
		//jpeg.Encode(picFile, img, nil)
		if err := png.Encode(picFile2, img); err != nil {
			log.Println("png.Encode error:", err)
		}
	}()

	if *bShowMap {
		DrawToConsole(&m)
	}
	log.Println("done w,h=", width, height, "maxColor=", maxColor, "nHills=", *nHills, "flowlen=0", "ridgelen=", ridge.length)
	select {}
}

func DrawToImg(img *image.RGBA, m *Topomap, w *WaterMap, maxColor float32, zoom *int, riverArrowScale *float64) {
	height := m.height
	width := m.width
	var tmpColor float32 = 1

	// 获取颜色模板
	var colorTplFile string = "image/color-tpl.png"
	cs := colorTpl(colorTplFile)
	//csmargin := 1
	cslen := len(cs) - 1 // - csmargin
	// 地图背景地形绘制
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = float32(m.data[x+y*width])
			//img.Set(x, y, color.RGBA{uint8(0xFF * tmpColor / maxColor), 0xFF, uint8(0xFF * tmpColor / maxColor), 0xFF})
			// 比例上色
			//img.Set(x, y, cs[int(float32(cslen)*(1.0-tmpColor/maxColor))])
			// 按值上色
			//img.Set(x, y, cs[cslen-int(tmpColor)])
			// 放大
			for zix := 0; zix < *zoom; zix++ {
				for ziy := 0; ziy < *zoom; ziy++ {
					img.Set(x**zoom+zix, y**zoom+ziy, cs[int(float32(cslen)*(tmpColor/maxColor))])
				}
			}
		}
	}
	// 绘制WaterMap
	tmpLakeColor := color.RGBA{0, 0xFF, 0xFF, 0xFF}
	for _, dot := range w.data {
		// 绘制积水
		if dot.h > 0 {
			//lineTo(img, int(dot.x)**zoom+*zoom/2-1, int(dot.y)**zoom+*zoom/2, int(dot.x)**zoom+*zoom/2+1, int(dot.y)**zoom+*zoom/2, tmpLakeColor, tmpLakeColor, 1.0)
			img.Set(int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2+1, int(dot.y)**zoom+*zoom/2+1, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2+1, int(dot.y)**zoom+*zoom/2, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2+1, int(dot.y)**zoom+*zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2-1, int(dot.y)**zoom+*zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2-1, int(dot.y)**zoom+*zoom/2, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2-1, int(dot.y)**zoom+*zoom/2+1, tmpLakeColor)
			img.Set(int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2+1, tmpLakeColor)
		}
	}

	// 绘制流动
	for di, dot := range w.data {
		// 如果是源头 则绘制白色
		if dot.dirPower > 0.0 {
			// 计算相对比例尺的高度
			tmpLevel := float32(m.data[di]) + float32(dot.h)
			tmpLevel = float32(cslen) * (tmpLevel / maxColor)
			// 防止越界
			if int(tmpLevel) >= len(cs) {
				tmpLevel = float32(len(cs) - 1)
			}
			if tmpLevel < 0 {
				tmpLevel = 0
			}
			// 下一点太远 放弃
			//nextX, nextY := dot.x+dot.xPower, dot.y+dot.yPower
			//if (nextX-int(dot.x))*(nextX-int(dot.x))+(nextY-int(dot.y))*(nextY-int(dot.y)) > 4 {
			//log.Println("the next is too far:", dot)
			//continue
			//}

			// 绘制流动方向 考虑缩放
			tmpColor := cs[int(tmpLevel)]
			if dot.xPower != 0 || dot.yPower != 0 {
				lineTo(img, int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2, int(dot.x)**zoom+*zoom/2+int(float32(*zoom)*dot.xPower), int(dot.y)**zoom+*zoom/2+int(float32(*zoom)*dot.yPower), color.RGBA{0, 0, 0xFF, 0xFF}, tmpColor, *riverArrowScale)
			}
			//log.Println("hasNext:", dot, w.data[dot.nextIdx])

			// 如果是源头 则绘制白色
			if len(dot.input) == 0 {
				//tmpColor = color.White
				img.Set(int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2, color.White)
			}
		}
	}
	// 绘制颜色模板
	for i := 0; i < len(cs); i++ {
		c := cs[len(cs)-i-1]
		for wi := 0; wi < 5; wi++ {
			img.Set(wi, i, c)
		}
	}
}

func DrawToConsole(m *Topomap) {
	width := m.width

	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~!@#$%^&*-=_+()[]{}<>\\/;:,.???????????????????????????????????????"
	for di, dd := range m.data {
		fmt.Printf("%c", str[dd])
		if di%width == (width - 1) {
			fmt.Printf("\n")
		}
	}
}

func DrawToHtml(w *WaterMap, m *Topomap) {

	footerHtml := "<table>"
	for wi := 0; wi < w.width; wi++ {
		footerHtml += "<tr>"
		for hi := 0; hi < w.height; hi++ {
			idx := hi*w.height + wi
			tdot := &w.data[hi*w.height+wi]
			footerHtml += fmt.Sprintf(`<td title="dir=%f hasdir=%v xpower=%f ypower=%f h=%d" style="width:1px;height:1px;background:rgb(0,%d,0)">&nbsp;&nbsp;</td>`,
				tdot.dir, tdot.dirPower, tdot.xPower, tdot.yPower, m.data[idx], m.data[idx]*10)
		}
		footerHtml += "</tr>"
	}
	footerHtml += "</table>"

	drawer.SetHomeDrawHandler(func(rw http.ResponseWriter) {
		rw.Write([]byte(footerHtml))
	})
	log.Printf("draw to html ok")
}
