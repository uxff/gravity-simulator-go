package main

// 计算对象到天体移动路径所在直线的垂心
func (o *Orb) CalcVertiDot(target *Orb) (vx, vy, vz float64) {
	// 斜率公式: k = -((x1-x0)(x2-x1)+(y2-y1)(y1-y0)+(z2-z1)(z1-z0))/((x2-x1)^2+(y2-y1)^2+(z2-z1)^2)
	// 垂点公式: xn=k(x2-x1)+x1 yn=k(y2-y1)+y1 zn=k(z2-z1)
	var x0, x1, x2, y0, y1, y2, z0, z1, z2 float64 = target.X, o.X, o.X - o.Vx, target.Y, o.Y, o.Y - o.Vy, target.Z, o.Z, o.Z - o.Vz
	k := -((x1-x0)*(x2-x1) + (y2-y1)*(y1-y0) + (z2-z1)*(z1-z0)) / ((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + (z2-z1)*(z2-z1))
	vx = k*(x2-x1) + x1
	vy = k*(y2-y1) + y1
	vz = k*(z2-z1) + z1

	return vx, vy, vz
}

// 判断移动对象是否穿过天体移动路径
func (o *Orb) IsThrough(target *Orb, dist float64) (bool, bool) {
	var isVertDistBigger, isSpanOn bool = false, false
	// 计算垂心距离
	verticalX, verticalY, verticalZ := o.CalcVertiDot(target)
	isVertDistBigger = ((verticalX-target.X)*(verticalX-target.X) + (verticalY-target.Y)*(verticalY-target.Y) + (verticalZ-target.Z)*(verticalZ-target.Z)) > MIN_CRITICAL_DIST*MIN_CRITICAL_DIST

	// 如果垂心距离target比临界半径大 则不相交
	// 如果垂心距离小，且与target形成的角度都是锐角，则相交
	// da^2 + do^2 > db^2 && db^2 + do^2 > da^2
	if !isVertDistBigger {
		oldVDistSquare := (o.X-o.Vx-target.X)*(o.X-o.Vx-target.X) + (o.Y-o.Vy-target.Y)*(o.Y-o.Vy-target.Y) + (o.Z-o.Vz-target.Z)*(o.Z-o.Vz-target.Z)
		isSpanOn = (oldVDistSquare+o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz) > (dist*dist) && (o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz+dist*dist) > oldVDistSquare
	}
	return isSpanOn, isVertDistBigger
}
