# go语言并行模拟万有引力天体运行

使用golang+js实现的万有引力模拟程序。(N体计算方式)

服务端使用go利用多核并行计算，支持memcache/redis/file存储计算数据，前后端使用websocket通信，前端使用ThreeJS框架，利用WebGL显示3D效果。

- 可模拟恒星系统，球状星团，星系，星流等
- 可设置中心超大质量天体(黑洞)
- 可合并多个集合，比如合并椭球星系和螺旋星系
- 可批量操作参数，比如放大分布范围，加快速度，增加质量，用来把球状星团放大到星系尺度等模拟
- 可保存和加载模拟的N体对象

效果图：

<img src="https://github.com/uxff/gravity-simulator-go/raw/master/image/gravity_sim_20170430204800.jpg">
<img src="https://github.com/uxff/gravity-simulator-go/raw/master/image/gravity_sim_20170430213314.jpg">
<img src="https://github.com/uxff/gravity-simulator-go/raw/master/image/gravity_sim_20170430210511.jpg">


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
$ go build go_server/calcserver/calc_server.go
#编译通信服务器
$ go build go_server/websocketserver/websocket_server.go
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
- 螺旋盘状分布
> $ ./calc_server --savekey thelist5 --init-orbs 1000 --config-arrange 0 --config-assemble 1 --config-wide 100000 --calc-times 100000
- x-y轴螺旋盘状再+z轴旋转
> ...
- 批量操作：y轴加速2倍，质量增加3倍，沿x轴左移4000单位
> $ ./calc_server --savekey thelist6 --calc-times 0 --moveexp 'vx=\*2&m=\*3&x=-4000'
- 将thelist2和thelist3合并，保存到thelist3中
> $ ./calc_server --loadkey thelist2 --savekey thelist3 --domerge


** 感想 **

- 初始时在一定空间随机分布一定数量天体，并在中心设置一个大质量天体(质量占所有天体质量的50%-90%)，随着时间变化，只有5%-10%左右会形成稳定轨道，而其他要么被撞击摧毁，要么逃逸。留下来的稳定轨道的类似行星天体，轨道趋于圆形(离心率低)的，数量更少，占稳定轨道天体数的5%-15%。这种轨道呈同心圆分布的，更少，呈同心圆分布超过3层的，在多次模拟中基本没出现过。

- 中心天体自转的方向，旋转速度是否对环绕它的仆从天体的运行轨道产生影响？(本模拟中不支持计算中心天体自转。) 因为现实中：太阳的行星们的轨道基本在同一黄道面；土星和木星的卫星们，基本也在它们的赤道平面上，尤其是它们的光环。

- 银河系的悬臂也基本在银盘这个平面上，而只有星系核心的部分才凸起。

- 中心天体自转对卫星天体运行轨道的影响，这种现象暂时没有模拟出来。(期望使用双黑洞模拟，算力有限，没有模拟出来，或许需要引力波理论的其他公式)

- 银河系属于漩涡星系，漩涡星系跟椭圆星系相比，是不是少了一个维度？椭圆星系是球状的，漩涡星系是盘状。漩涡星系是不是被扔了一个二向箔，三维中的一个维度被扭曲压缩，形成悬臂，星系核附近扭曲的更厉害。可能没有二向箔，星系核中心特殊的结构可能会发生类似二向箔作用的力量和演化。

- 盘状的漩涡星系，是从球状的矮椭圆星系演化来的悬臂，还是从核球喷出气体流，气体流逐渐形成恒星，随着核球旋转形成漩涡形的悬臂？

- 盘状的漩涡星系的悬臂，是呈密度波分布，密度波是核心的双黑洞，或者多黑洞相互高速环绕旋转形成的吗？

- 悬臂结构的银河系，悬臂转1圈后，形状是会变化的。这种变化证明星系核的质量在变化，起码在转一圈的周期内，有巨大变化。这种变化类似黑洞的吞噬，质量进去了无法出来，就是一部分在外围轨道的，转到内部后无法转出来，形成一种不平衡。而椭圆星系却有一种平衡。

- 一个漩涡结构的星系，其形状并不能稳定维持很长时间以上，即便寿命很长，他每经过10亿年的样子是完全不一样的。

- 银河系从诞生到现在，自转不超过50圈。这么少的圈数，还很难形成稳定的沉淀结构，比如椭球星系一样的结构。

- 一个星系状态达到椭圆星系以后，可以上百亿年保持这种形状。

- 在一个椭圆星系中的文明，也许是悲哀的，虽然有稳定的条件，但是没有新星的出现催化生命的进化，提供丰富的原材料。

- 我们人类很庆幸出现在一个漩涡星系中。

- 生命出现在一个动态变化，似乎不平衡，但又有些安静的角落里。比如太阳系，跟在银河系里，但又不在悬臂上。

- 环形星系，车轮星系，外部的环应该是迅速扩大的。像火焰有里往外蹿烧一样，迅速把星系周围的气体燃烧起来。

- 悬臂是否预示着一种高密度物质的运动轨迹？一种能量波？
- 悬臂的一头是否被拉扯进中心而无法出来？
- 天体之间还有其他作用力，非万有引力的作用力，来产生悬臂这种不均衡的能量波。
- 天体本身的自传，温度，内部流体动能等，是否是碰撞带来的？简介影响了碰撞后的动能？
- 天体本身的自传，是否会与其他天体的自传形成共振？共振是否导致了大部分周边天体朝一个方向运动围绕中心天体运动。
- 悬臂及星系如何形成猜测1：中心天体旋转导致旋转面维度拉长。外围物体加速度不变，只是掉落入中心天体的路径被拉长，从外部看起来其轨迹变成螺旋运动。
- 悬臂及星系如何形成猜测2：中心天体吸积盘层面扩大扰动氢分子云导致盘面维度恒星生成。
- 悬臂及星系如何形成猜测3：原来是球状，中心天体两极喷流，喷走了上下两侧天体，只剩下盘面。但是有BUG：喷流只在一个极小的角度，喷流周围应该还有大量天体存在。
- 星系如何形成猜测：为什么凭空出现了星系？一个种子吸积盘产生了连锁效应，将周边紧密氢云点燃，促成了恒星形成连锁反应？
- 悬臂生成猜测：悬臂由[具有另一个周期的]中心天体喷流向周围气体云产生。气体云由静态逐渐被点燃产生造星反应。另一个周期是指，这个喷流在缓慢摇摆，与中心天体的自转周期不一样。
- 悬臂生成猜测条件：中心天体的自转方向与观察到的悬臂方向是一致的？
- 宇宙本身处在一个很高的势能下。这个势能下有大量的氢，未被消耗为重元素。至少可观测宇宙是这样。
- 大量的未被凝结为恒星的氢，质量甚至超过已知宇宙天体总和。
- 可能存在势能很低的宇宙，这个宇宙下万有引力失控，或者氢过度消耗，过多老年恒星，无法形成
- 目前观测到的宇宙膨胀，是造物主设计以用来对抗万有引力。
- 万有引力只有引力没有斥力，最终其他力(电磁力，强核力，弱核力)无法对抗引力导致物理世界最终将崩溃。
- 电磁力之库伦力方程，描述的力与距离的平方成反比。证明电磁力作用与二维空间。
- 分子间范德华力作用，与距离3次方或6次方，甚至12次方有关系。似乎可以看作是期间有其他维度的作用力。
- 带点粒子的旋转会导致其产生电磁力。带质量物体旋转是否会有类似的效应？如果没有，如何解释漩涡星系？
- 悬臂及漩涡计算考虑：引力如果以引力波传递，则中心天体的质量增加，对外围施加的引力变化，是有延迟的。是否与形成悬臂结构有关。似乎关系不大。反推：如果中心天体质量稳定，则悬臂会逐渐解散，实际并没观察到这一现象。
- 计算考虑：过高的速度，将拉扯天体内部稳定性，可能使其内部温度升高，而升高的温度则吸收一部分速度。
- 计算考虑：碰撞将不能告诉弹走对方，应该吸收速度，增加内能，减少质量。温度越高，别人撞击它时，对自己速度变化影响更小。温度会挥发。


## 其他

- [js gravity simulator](https://github.com/uxff/gravity-simulator)
- [gravity simulator mpi](https://github.com/uxff/gravity-simulator-mpi)

cps:

i7 4720H | 1.6e+08
i7 8750H | 6.5e+08
i5 10300H | 5.12e+08
r7 4800h | 6.994020e+08 ~ 8.79e+08
i7 14650HX | 2.009911e+09
