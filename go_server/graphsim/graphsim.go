package main

import (
	"fmt"
	"image"
	"image/color"
	//"image/draw"
	"flag"
	//"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"time"
)

type WaterDot struct {
	x       float32
	y       float32
	dir     float64
	h       int   // 积水高度，产生积水不参与流动，流动停止
	q       int   // 流量 0=无 历史流量
	input   []int // 流入坐标
	nextIdx int
	hasNext bool // 有下一个方向
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
	this.lastIdx = x + y*w.width
	this.List[0] = this.lastIdx
	this.lastDot = &w.data[this.lastIdx]
	this.lastDot.x = float32(x) + 0.5
	this.lastDot.y = float32(y) + 0.5
	this.lastDot.q = 1
	this.length = 1
	theDir := rand.Float64() * math.Pi * 2.0
	this.lastDot.dir = theDir
	this.step = 1
	//fmt.Println("new dir:", theDir, this.lastDot)

	//w.data[x+y*w.width] = *this.lastDot
}

// 随机洒水法：
/*
	随机在地图中选择点，并滴入一滴水，记录水位+1，尝试计算流出方向(判断旁边的水流方向)
	如果没有流出方向，水位+1；如果有流出方向，按方向滴入下一位置,本地水位-1
	向WaterMap中的某个坐标注水
	水尝试找一个流动方向
	注水，选择方向，流动耦合
*/
func (w *WaterMap) InjectWater(pos int, m *Topomap) bool {
	if pos >= len(w.data) {
		return false
	}
	var curX, curY int = pos % w.width, pos / w.height
	var curDot *WaterDot = &w.data[pos]

	// 过量退出，否则栈溢出
	if curDot.q > 255 {
		// 理应断开与上下游关系，产生积水
		if curDot.hasNext {
			if w.data[curDot.nextIdx].q > 240 {
				fmt.Println("seems flow in circle, cut me->next, me=:", curDot)
				curDot.h++
				curDot.hasNext = false
				for _, itsInputIdx := range curDot.input {
					w.data[itsInputIdx].hasNext = false
				}
				curDot.input = []int{}
			} else {
				fmt.Println("none circle exceed occur? next q=", w.data[curDot.nextIdx].q)
			}
		}
		fmt.Println("too much quantity me=", curDot)
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
			fmt.Println("lower first curDot,tx,ty=", curDot, tx, ty)
			break
		} else if lowest > int(m.data[pos]) {
			curDot.hasNext = false
			fmt.Println("desire to go out", curDot)
			break
		}

		var pit1 float64 = rand.Float64() - rand.Float64()
		rollDir = pit1 * pit1 * pit1 * (math.Pi/1.2 + float64(i)/10.0)
		//rollDir = rand.Float64() * math.Pi * 2.0
		theDir = rollDir + curDot.dir

		allvx, allvy = (math.Cos(theDir)), (math.Sin(theDir))
		// 碰到边界
		tx, ty = int(float64(curDot.x)+allvx), int(float64(curDot.y)+allvy)
		if tx < 0 || ty < 0 || int(tx) > w.width-1 || int(ty) > w.height-1 {
			fmt.Println("over bound tx,ty:", tx, ty, "o=", curDot, "i=", i)
			//continue
			// 任其流出地图外，不再让其流回来
			//curDot.h--
			return false
		}
		if tx == curX && ty == curY {
			// 内部移动后还在内部，将下次的基点延长
			curDot.x, curDot.y = curDot.x+float32(allvx)/2, curDot.y+float32(allvy)/2
			//fmt.Println("inner step")
			continue
		}
		// 计算地形落差 地形较高 不允许流向高处
		tIdx = ty*m.width + tx

		assumeFall := (int(m.data[tIdx])*1 + w.data[tIdx].h) - (int(m.data[pos])*1 + curDot.h)
		if assumeFall >= 1 {
			fmt.Println("flow up, fall,dot,tx,ty=", assumeFall, curDot, tx, ty, "i=", i)
			continue
		}

		// 对方的nextIdx不能是me
		if w.data[tIdx].hasNext && w.data[tIdx].nextIdx == pos {
			//continue
			// 撤销对方指向me的next，如果我方地形高
			if assumeFall < 0 {
				fmt.Println("discard target next direct me,target=", curDot, w.data[ty*m.width+tx])
				w.data[tIdx].hasNext = false
			} else {
				fmt.Println("cannot flow to target because its next is me: me=", curDot, "target=", w.data[ty*m.width+tx], "i=", i)
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
				fmt.Println("become cross:x,y,tx,ty=", curX, curY, tx, ty, "redir to x,y=", near2x, near2y)
				tx, ty = near2x, near2y
			} else if w.data[near2x+near2y*w.width].hasNext && w.data[near2x+near2y*w.width].nextIdx == (near1x+near1y*w.width) {
				// 下游是1
				fmt.Println("become cross:x,y,tx,ty=", curX, curY, tx, ty, "redir to x,y=", near1x, near1y)
				tx, ty = near1x, near1y
			}
		}

		curDot.hasNext = true
		break
	}

	// 选择成功则流入
	if curDot.hasNext {
		fmt.Println("got dir ok: rollDir=", rollDir, "pos xy=", pos, curX, curY, "target x,y:", tx, ty)
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
		fmt.Println("cannot flow anywhere: curDot=", curDot)
		// 积水
		curDot.h++
		// 切断curDot跟下游的关系 本来就是断的
		//curDot.hasNext = false
		// 切断curDot跟上游的关系
		for _, itsInputIdx := range curDot.input {
			w.data[itsInputIdx].hasNext = false
		}
		curDot.input = make([]int, 0)
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
		fmt.Println("png.decode err when read colorTpl:", err)
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
	d.x, d.y = (d.x+inputX)/2, (d.y+inputY)/2
}

// 此函数固定返回8个边界点，可能包含超出地图边界的点
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

/*获取周围最低的点 最低点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getLowestNeighbor(w *WaterMap, m *Topomap) (lowest int, lowestVal struct{ x, y int }) {
	pos := d.getNeighbors(w)
	// 原理： highMap[high] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range pos {
		if nei.x < 0 || nei.x > w.width-1 || nei.y < 0 || nei.y > w.height-1 {
			// 超出地图边界的点
			continue
		}
		high := int(m.data[nei.x+nei.y*w.width])
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	lowest = 1000
	for k, _ := range highMap {
		if k < lowest {
			lowest = k
		}
	}
	//fmt.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[lowest]) == 0 {
		return lowest, lowestVal
	}
	return lowest, highMap[lowest][rand.Int()%len(highMap[lowest])]
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	var times = flag.Int("flow", 5, "flow move times")
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
	var river, ridge FlowList
	w.Init(width, height)
	m.Init(width, height)
	river.Init(width/2, height/2, &w, *times+1)
	ridge.Init(width/2, height/2, &w, *nRidge+1)
	ridge.step = float64(*ridgeStep)

	if _, derr := os.Open(*outdir); derr != nil {
		fmt.Println("dir seems not exist:", *outdir, derr)
		if cerr := os.Mkdir(*outdir, os.ModePerm); cerr != nil {
			fmt.Println("os.mkdir:", *outdir, cerr)
			//return
		}
	}
	//picFile, _ := os.Create(*outname + ".jpg")
	filename := fmt.Sprintf("%s-%s", *outname, time.Now().Format("20060102150405"))
	picFile2, _ := os.Create(*outdir + "/" + filename + ".png")
	defer picFile2.Close()

	img := image.NewRGBA(image.Rect(0, 0, width**zoom, height**zoom))

	// 随机n个圆圈 累加抬高
	rings := make([]Ring, *nHills)
	for ri, _ := range rings {
		r := &rings[ri]
		r.x, r.y, r.r, r.h = (rand.Int() % width), (rand.Int() % height), (rand.Int()%(*hillWide) + 1), (rand.Int()%(5) + 2)
		r.tiltDir, r.tiltLen = rand.Float64()*math.Pi, (rand.Int()%10)+1
	}

	// make ridge 生成ridge的痕迹
	//for i := 1; i < *nRidge; i++ {
	//	ridge.Move2(&m, &w)
	//}
	// 转换痕迹为ridge 为每个环分配随机半径
	ridgeRings := make([]Ring, ridge.length)
	for ri := 0; ri < int(ridge.length); ri++ {
		r := &ridgeRings[ri]
		r.x, r.y, r.r = ridge.List[ri]%width, ridge.List[ri]/width, (rand.Int() % (*ridgeWide))
	}
	fmt.Println("ridgeRings=", ridgeRings)

	// 生成地图 制造地形
	var tmpColor, maxColor float32 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = 0
			// 收集ridgeRings产生的attitude
			for _, r := range ridgeRings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				if distM <= r.r*r.r {
					tmpColor++
					//tmpColor += float32(distM) / float32(r.r*r.r) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//fmt.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}
			// 收集rings产生的attitude
			for _, r := range rings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				//rn := float64(r.tiltLen)*math.Sin(r.tiltDir-math.Atan2(float64(y), float64(y))) + float64(r.r)	// 尝试倾斜地图中的圆环 尝试失败
				rn := (r.r)
				if distM <= int(rn*rn) {
					// 产生的ring中间隆起
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(float64(distM)/float64((rn*rn))))
					//tmpColor += float32(distM) / float32(rn*rn) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//fmt.Println("color fill x,y,r,c=", x, y, r, tmpColor, "r=", r)
				}
			}

			m.data[x+y*width] = uint8(tmpColor) //+ uint8(rand.Int()%2) //int8(width - x)
		}
	}
	maxColor *= 3

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

	// 随机洒水
	for i := 0; i < *times; i++ {
		idx := rand.Int() % len(w.data)
		//x, y := idx%w.width, idx/w.width
		//w.data[idx].x, w.data[idx].y = float32(x)+0.5, float32(y)+0.5
		w.InjectWater(idx, &m)
	}
	// 顺序洒水
	//	for i := 0; i < len(w.data); i++ {
	//		if i%7 == 0 {
	//			w.InjectWater(i, &m)
	//		}
	//	}
	// 中间洒水
	//		idx :=
	//		//x, y := idx%w.width, idx/w.width
	//		//w.data[idx].x, w.data[idx].y = float32(x)+0.5, float32(y)+0.5
	w.InjectWater(w.width/2+w.width*w.height/2, &m)

	// 绘制river
	// 使用zoomstep lineTo
	//var stepStartX, stepStartY, stepDestX, stepDestY int
	//for i, dot := range river.List {
	//	stepStartX, stepStartY = stepDestX, stepDestY
	//	stepDestX, stepDestY = dot%width, dot/width
	//	if i == 0 || stepDestX == 0 && stepDestY == 0 {
	//		continue
	//	}
	//	// lineTo
	//	//img.Set(int(x)**zoom+*zoom/2, int(y)**zoom+*zoom/2, color.RGBA{0, 0, 0xFF, 0xFF})
	//	lineTo(img, int(stepStartX)**zoom+*zoom/2, int(stepStartY)**zoom+*zoom/2, int(stepDestX)**zoom+*zoom/2, int(stepDestY)**zoom+*zoom/2, color.RGBA{0, 0, 0xFF, 0xFF})
	//}
	//lineTo(img, 100, 200, 200, 200, color.RGBA{0, 0, 0xFF, 0xFF})

	// 绘制WaterMap
	for di, dot := range w.data {
		// 绘制积水
		if dot.h > 0 {
			tmpColor := color.RGBA{0, 0xFF, 0xFF, 0xFF}
			lineTo(img, int(dot.x)**zoom+*zoom/2-1, int(dot.y)**zoom+*zoom/2, int(dot.x)**zoom+*zoom/2+1, int(dot.y)**zoom+*zoom/2, tmpColor, tmpColor, 1.0)
		}
		// 绘制流动
		if dot.hasNext {
			tmpLevel := float32(m.data[di]) + float32(dot.h)
			tmpLevel = float32(cslen) * (tmpLevel / maxColor)
			if int(tmpLevel) >= len(cs) {
				tmpLevel = float32(len(cs) - 1)
			}
			if tmpLevel < 0 {
				tmpLevel = 0
			}
			nextX, nextY := dot.nextIdx%w.width, dot.nextIdx/w.height
			if (nextX-int(dot.x))*(nextX-int(dot.x))+(nextY-int(dot.y))*(nextY-int(dot.y)) > 4 {
				//fmt.Println("the next is too far:", dot)
				continue
			}
			tmpColor := cs[int(tmpLevel)]
			lineTo(img, int(dot.x)**zoom+*zoom/2, int(dot.y)**zoom+*zoom/2, (dot.nextIdx%w.width)**zoom+*zoom/2, (dot.nextIdx/w.width)**zoom+*zoom/2, color.RGBA{0, 0, 0xFF, 0xFF}, tmpColor, *riverArrowScale)
			//fmt.Println("hasNext:", dot, w.data[dot.nextIdx])
		}
	}

	// 绘制颜色模板
	for i := 0; i < len(cs); i++ {
		c := cs[len(cs)-i-1]
		for wi := 0; wi < 5; wi++ {
			img.Set(wi, i, c)
		}
	}

	// 输出图片文件
	//jpeg.Encode(picFile, img, nil)
	if err := png.Encode(picFile2, img); err != nil {
		fmt.Println("png.Encode error:", err)
	}

	if *bShowMap {
		//fmt.Println("map=", m)
		str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~!@#$%^&*-=_+()[]{}<>\\/;:,.???????????????????????????????????????"
		for di, dd := range m.data {
			fmt.Printf("%c", str[dd])
			if di%width == (width - 1) {
				fmt.Printf("\n")
			}
		}
		//fmt.Println("wmap=", w)
	}
	fmt.Println("done w,h=", width, height, "maxColor=", maxColor, "nHills=", *nHills, "flowlen=", river.length, "ridgelen=", ridge.length)
}
