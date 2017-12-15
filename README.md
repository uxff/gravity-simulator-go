# go语言并行模拟万有引力天体运行

使用golang+js实现的万有引力模拟程序。(N体计算方式)

服务端使用go利用多核并行计算，支持memcache/redis/file存储计算数据，前后端使用websocket通信，前端使用ThreeJS框架，利用WebGL显示3D效果。

- 可模拟恒星系统，球状星团，星系，星流等
- 可设置中心超大质量天体(黑洞)
- 可合并多个集合，比如合并椭球星系和螺旋星系
- 可批量操作参数，比如放大分布范围，加快速度，增加质量，用来把球状星团放大到星系尺度等模拟
- 可保存和加载模拟的N体对象

效果图：

<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430204800.jpg">
<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430213314.jpg">
<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430210511.jpg">


## 思路：

有两个程序,calc_server和websocket_server  
calc_server -> 计算数据保存到file/mc/redis  
浏览器查看 -> websocket_server ->读取file/mc/redis的数据  

查看需要浏览器支持WebGL,比如chrome/firefox。
ie各个版本/360急速/360安全浏览器无法查看。
chrome下支持100万粒子查看,并保持50+fps的流畅度,内存占用0.9-1.8G浮动。

为了达到良好体验，浏览器端操作系统最好4G以上内存，64位操作系统，使用redis通信，使用64位chrome。

## 编译：
Mac/Linux/Windows
```
#编译计算服务器
$ go build go_server/calc_server.go
#编译通信服务器
$ go build go_server/websocket_server.go
```

## 使用方法：

```
#step 1.初始化数据
#这里初始化1000个，设置初始速度，初始质量，中心恒星质量
$ ./calc_server -init-orbs 1000 -config-velo 0.05 -config-mass 100 -bigmass 18000
#step 2.执行演化10000次 可能会时间比较长 最后开新窗口执行下面
$ ./calc_server -calc-times 10000
#step 3.运行通信服务器
$ ./websocket_server -addr 192.168.12.150:8081
#step 4.使用chrome或firefox打开websocket_server对应的地址http://192.168.12.150:8081/index.html查看运行情况(需要浏览器支持WebGL和websocket)
#如果step 2步执行完毕，会自动保存数据到memcache，mc默认使用127.0.0.1:11211。
#继续执行step 2步，回到浏览器查看效果。

```
使用 calc_server -h 和 websocket_server -h 查看对应的服务端参数和用法。


## calc_server 参数说明：
```
  -bigmass float
    	config of big mass orb, like a blackhole in center, 0 means no bigger (default 15000)
        中心天体质量 0表示没有大质量天体 默认15000
  -bignum int
    	config of number of big mass orbs, generally center has 1 (default 1)
        中心大质量天体数量 通常为1个
  -bigstyle int
    	config of big mass orb distribute style: 0=center,1=outer edge,2=middle of one radius,3=random
        中心大质量天体分布方式 0=中央 1=外边缘 2=半径的中点 3=随机
  -calc-times int
    	how many times calc (default 100)
        计算次数
  -config-arrange int
    	init style of orbs arrangement: 0=line,1=cube,2=disc,3=sphere (default 3)
        初始化时排列方式 0=线性(在线周围会有浮动) 1=立方体 2=盘状 3=球形
  -config-assemble int
    	init style of orbs aggregation: 0=avg,1=ladder,2=variance,3=4th power (default 2)
        初始化聚集方式 0=平均分布 1=梯形分布 2=平方差 3=4次方差
  -config-cpu int
    	how many cpu u want use, 0=all
        使用多少cpu
  -config-mass float
    	init mass of orbs (default 10)
        初始化天体质量范围 默认10
  -config-velo float
    	init velo of orbs (default 0.005)
        初始化天体速度范围 默认0.005 不要太快，快了中心天体束缚不住
  -config-wide float
    	init wide of orbs (default 1000)
        初始化分布宽度范围 默认1000
  -domerge
    	merge from loadkey to savekey if true, replace if false
        合并loadkey中的天体列表到savekey中
  -init-orbs int
    	how many orbs init, do init when its value >1
        初始化天体数量 大于1将初始化 不传将使用loadkey的天体列表数据
  -loadkey string
    	key name to load, like key of memcache, or filename in save dir, use savekey if no given
        要加载的天体列表名，可以是存在memcache的key,可以是文件名 没指定则使用savekey指定的值
  -loadpath string
    	where to load, support mc/file/redis
        从哪里加载天体列表，支持memcache,file,redis. 没指定使用savepath指定的值
	like: file://./filecache/, use savepath if no given
    举例：文件方式当前目录下的filecache目录下 file://./filecache/ ;memcache方式 mc://127.0.0.1:11211 ;redis方式 redis://127.0.0.1:6179
  -moveexp string
    	move expression, like: x=-150&vy=+0.01&m=+20 only position,velo,mass
        使用表达式批量移动列表数据 只支持坐标，速度，质量 不支持数量,加速度,id
  -save-duration int
    	save to savepath per millisecond, 100 means 100ms (default 500)
        保存间隔 毫秒 1500=计算时每1.5秒保存一次 保存过于频繁影响计算速度，过慢影响浏览器查看效果
  -savekey string
    	key name to save, like key of memcache, or filename in save dir (default "thelist1")
        要保存的天体列表名
  -savepath string
    	where to save, support mc/file/redis
        在哪里保存天体列表数据，格式同loadpath
	like: file://./filecache/ (default "mc://127.0.0.1:11211")
  -showlist
    	show orb list and exit
        显示列表数据，并退出，不继续计算
```

## 模拟示例
- 线状分布，靠中点聚拢分布
> $ ./calc_server --savekey thelist2 --init-orbs 1000 --config-arrange 0 --config-assemble 2 --config-wide 100000 --calc-times 0
- 盘状分布，均衡分布
> $ ./calc_server --savekey thelist3 --init-orbs 1000 --config-arrange 2 --config-assemble 0 --config-wide 100000 --calc-times 0
- 球状分布，4次方差中心聚集
> $ ./calc_server --savekey thelist4 --init-orbs 1000 --config-arrange 3 --config-assemble 3 --config-wide 100000 --calc-times 0
- 盘状双黑洞
> $ ./calc_server --savekey thelist5 --init-orbs 1000 --bignum 2 --bigstyle 2 --config-arrange 0 --config-assemble 2 --config-wide 100000 --calc-times 0
- 模拟螺旋盘状分布的集合
> ...
- 模拟x-y轴螺旋盘装再+z轴旋转
> ...

** 感想 **

- 初始时在一定空间随机分布一定数量天体，并在中心设置一个大质量天体(质量占所有天体质量的50%-90%)，随着时间变化，只有5%-10%左右会形成稳定轨道，而其他要么被撞击摧毁，要么逃逸。留下来的稳定轨道的类似行星天体，轨道趋于圆形(离心率低)的，数量更少，占稳定轨道天体数的5%-15%。这种轨道呈同心圆分布的，更少，呈同心圆分布超过3层的，在多次模拟中基本没出现过。

- 中心天体自转的方向，旋转速度是否对环绕它的仆从天体的运行轨道产生影响？(本模拟中不支持计算中心天体自转。) 因为现实中：太阳的行星们的轨道基本在同一黄道面；土星和木星的卫星们，基本也在它们的赤道平面上，尤其是它们的光环。

- 银河系的悬臂也基本在银盘这个平面上，而只有星系核心的部分才凸起。

- 中心天体自转对卫星天体运行轨道的影响，这种现象暂时没有模拟出来。

- 银河系属于漩涡星系，漩涡星系跟椭圆星系相比，是不是少了一个维度？椭圆星系是球状的，漩涡星系是盘状。漩涡星系是不是被扔了一个二向箔，三维中的一个维度被扭曲压缩，形成悬臂，星系核附近扭曲的更厉害。可能没有二向箔，星系核中心特殊的结构可能会发生类似二向箔作用的力量和演化。

- 盘状的漩涡星系，是从球状的矮椭圆星系演化来的悬臂，还是从核球喷出气体流，气体流逐渐形成恒星，随着核球旋转形成漩涡形的悬臂？

- 盘状的漩涡星系的悬臂，是呈密度波分布，密度波是核心的双黑洞，或者多黑洞相互高速环绕旋转形成的吗？


## 其他

- [js gravity simulator](https://github.com/uxff/gravity-simulator)
- [gravity simulator mpi](https://github.com/uxff/gravity-simulator-mpi)
