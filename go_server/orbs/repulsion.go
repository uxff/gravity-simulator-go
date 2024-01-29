package orbs

const (
	// RepulsionG 斥力常数
	RepulsionG = G * 2
)

// 斥力与距离的三次方成反比

// 计算天体与目标的引力 与距离三次方成反比
func (o *Orb) CalcRepulsionF(target *Orb, dist float64) Acc {
	var a Acc

	// 万有斥力
	a.A = target.Mass / (dist * dist * dist * dist) * RepulsionG
	a.Ax = -a.A * (target.X - o.X) / dist
	a.Ay = -a.A * (target.Y - o.Y) / dist
	a.Az = -a.A * (target.Z - o.Z) / dist

	return a
}

func (a *Acc) add(ta *Acc) {
	a.Ax += ta.Ax
	a.Ay += ta.Ay
	a.Az += ta.Az
}
