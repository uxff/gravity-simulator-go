var calcDist, calcGravity, calcOrbs;
var CalcUnit = {
    key : null,
    orbList : null,
    feedList : null,
    stage : 0,
    tryGetOrbFailedTimes : 0,
    tryGetFeedFailedTimes : 0,
    loopFeed : false,
    setOrbList: function(orbList) {
        if (orbList instanceof Array) {
            this.orbList = orbList;
            this.tryGetOrbFailedTimes = 0;
        } else {
            this.tryGetOrbFailedTimes++;
        }
    },
    setFeedList: function(feedList) {
        if (feedList instanceof Array) {
            this.feedList = feedList;
            this.tryGetFeedFailedTimes = 0;
        } else {
            this.tryGetFeedFailedTimes++;
        }
    },
    consume: function() {
        if (!this.orbList instanceof Array) {
            console.log('illegal this.orbList:', this.orbList);
            if (this.tryGetOrbFailedTimes < 10) {
                this.reloadOrbList();
            }
        } else if (!this.feedList instanceof Array) {
            console.log('illegal this.feedList:', this.feedList);
            if (this.tryGetFeedFailedTimes < 10) {
                this.reloadFeedList();
            }
        } else {
            //console.log(this.feedList);
            for (i in this.feedList) {
                let crashedBy = calcOrbs(this.feedList[i], this.orbList);
                let o = this.orbList[this.feedList[i]];
                let content = 'cmd=recvorb&k='+this.key+'&idx='+this.feedList[i]
                    +'&crashedBy='+crashedBy+'&o='+JSON.stringify(o)+'&stage='+this.stage;
                MyWebsocket.doSend(content);
            }
            console.log('i have calced ',this.feedList,' stage='+this.stage);
        //if (this.loopFeed) {
            this.reloadFeedList();
        //}
        }
    },
    reloadOrbList: function() {
        let content = 'cmd=orbs&k=thelist1';
        MyWebsocket.doSend(content);
    },
    reloadFeedList: function() {
        let content = 'cmd=taketask&k=thelist1&calcnum=10';
        MyWebsocket.doSend(content);
    },
    start: function() {
        // get orb list
        //this.reloadOrbList();
        // get uncalcedorb
        //this.reloadFeedList();
        // calc one
        this.consume();
    }
};

/*计算入口*/
calcOrbs = function(orbId, orbList) {
    if (orbList == undefined) {
        console.log('orbList is not already');
        return false;
    }
    if (orbId >= orbList.length) {
        console.log('orbList as no id: '+orbId);
        return false;
    }
    let o = orbList[orbId];
    if (o.st != 1 || o.m == 0) {
        console.log('orb status not ok:'+orbId, o);
        return false;
    }
    let crashedBy = -1;
    let gAll = {x:0,y:0,z:0};
    for (let i in orbList) {
        let ta = orbList[i];
        if (ta.id == o.id || ta.st != 1 || o.st != 1) {
            continue;
        }
        let dist = calcDist(o, ta);
        let isTooNearly = dist*dist < 2*2;
        let isMeRipped = dist < Math.sqrt(o.vx*o.vx+o.vy*o.vy+o.vz*o.vz) * 8
        if (isTooNearly || isMeRipped) {
            if (o.m < ta.m) {
                crashedBy = i;
                o.st = 2;
            }
        } else {
            let gTmp = calcGravity(o, ta, dist);
            gAll.x += gTmp.x;
            gAll.y += gTmp.y;
            gAll.z += gTmp.z;
        }
    }
    //o.crashedBy = crashedBy;
    //return gAll;
    if (crashedBy >= 0) {
        
    } else {
        o.x += o.vx;
        o.y += o.vy;
        o.z += o.vz;
        o.vx += gAll.x;
        o.vy += gAll.y;
        o.vz += gAll.z;
    }
    //return o;
    return crashedBy;
}
calcDist = function(o, target) {
    return Math.sqrt((o.x-target.x)*(o.x-target.x) + (o.y-target.y)*(o.y-target.y) + (o.z-target.z)*(o.z-target.z));
}
calcGravity = function(o, target, dist) {
    let a = target.m / (dist*dist) * G;
    let aAll = {x:0,y:0,z:0};
    aAll.x = - a * (o.x - target.x) / dist;
    aAll.y = - a * (o.y - target.y) / dist;
    aAll.z = - a * (o.z - target.z) / dist;
    return aAll;
}


