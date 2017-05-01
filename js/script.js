/**
 *
 * WebGL With Three.js - Lesson 1
 * http://www.script-tutorials.com/webgl-with-three-js-lesson-1/
 *
 * Licensed under the MIT license.
 * http://www.opensource.org/licenses/mit-license.php
 * 
 * Copyright 2013, Script Tutorials
 * http://www.script-tutorials.com/
 */

var colors = [
    0xFF62B0,
    0x9A03FE,
    0x62D0FF,
    0x48FB0D,
    0xDFA800,
    0xC27E3A,
    0x990099,
    0x9669FE,
    0x23819C,
    0x01F33E,
    0xB6BA18,
    0xFF800D,
    0xB96F6F,
    0x4A9586
];
var particleLight;
var ticker = 0;
var zoomBase = 1.0, zoomStep = Math.sqrt(2.0);// = document.getElementById('zoom').value;
var NUM_PARTICLES = 0;
var MIN_DIST = 2.0;
var G = 0.000025;

var enableCalc = false;

var lesson1 = {
    scene: null,
    camera: null,
    renderer: null,
    container: null,
    controls: null,
    clock: null,
    stats: null,
    orbList: [],// new Map();
    dataList: null,// new Map();
    isInited: false,

    init: function() { // Initialization

        // create main scene
        this.scene = new THREE.Scene();

        var SCREEN_WIDTH = window.innerWidth,
            SCREEN_HEIGHT = window.innerHeight;

        // prepare camera
        var VIEW_ANGLE = 45, ASPECT = SCREEN_WIDTH / SCREEN_HEIGHT, NEAR = 2, FAR = 1000000;
        this.camera = new THREE.PerspectiveCamera( VIEW_ANGLE, ASPECT, NEAR, FAR);
        this.scene.add(this.camera);
        this.camera.position.set(1500, 3000, 3000);
        this.camera.lookAt(new THREE.Vector3(0,0,0));

        // prepare renderer
        this.renderer = new THREE.WebGLRenderer({antialias:true, alpha: false});
        this.renderer.setSize(SCREEN_WIDTH, SCREEN_HEIGHT);
        this.renderer.setClearColor(0xffffff);

        this.renderer.shadowMapEnabled = true;
        this.renderer.shadowMapSoft = true;

        // prepare container
        this.container = document.createElement('div');
        document.body.appendChild(this.container);
        this.container.appendChild(this.renderer.domElement);

        // events
        THREEx.WindowResize(this.renderer, this.camera);

        // prepare controls (OrbitControls)
        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        this.controls.target = new THREE.Vector3(0, 0, 0);

        // prepare clock
        this.clock = new THREE.Clock();

        // prepare stats
        this.stats = new Stats();
        this.stats.domElement.style.position = 'absolute';
        this.stats.domElement.style.bottom = '0px';
        this.stats.domElement.style.zIndex = 10;
        this.container.appendChild( this.stats.domElement );

        // 坐标系
        var axisHelper = new THREE.AxisHelper(1000); // 500 is size
        this.scene.add(axisHelper);

        // add directional light
        var dLight = new THREE.DirectionalLight(0xffffff);
        dLight.position.set(1, 1000, 1);
        dLight.castShadow = true;
        dLight.shadowCameraVisible = false;//true;
        dLight.shadowDarkness = 0.52;
        dLight.shadowMapWidth = dLight.shadowMapHeight = 1000;
        this.scene.add(dLight);

        //// add particle of light, show the position of light
        //particleLight = new THREE.Mesh( new THREE.SphereGeometry(10, 10, 10), new THREE.MeshBasicMaterial({ color: 0x44ff44 }));
        //particleLight.position = dLight.position;
        //this.scene.add(particleLight);

        // add simple ground
        //var groundGeometry = new THREE.PlaneGeometry(1000, 1000, 1, 1);
        //ground = new THREE.Mesh(groundGeometry, new THREE.MeshLambertMaterial({
        //    color: this.getRandColor()
        //}));
        //ground.position.y = 0;
        //ground.rotation.x = - Math.PI / 2;
        //ground.receiveShadow = true;
        //this.scene.add(ground);

        // add sphere shape
        //var sphere = new THREE.Mesh(new THREE.SphereGeometry(70, 32, 32), new THREE.MeshLambertMaterial({ color: this.getRandColor() }));
        //sphere.rotation.y = -Math.PI / 2;
        //sphere.position.x = 100;
        //sphere.position.y = 150;
        //sphere.position.z = 300;
        //sphere.castShadow = sphere.receiveShadow = true;
        //this.scene.add(sphere);
        //console.log(sphere);

        //this.orbList = new Map();
    },
    getRandColor: function() {
        return colors[Math.floor(Math.random() * colors.length)];
    },
    initOrbs: function(list) {
        if (list==undefined) {
            return false;
        }
        NUM_PARTICLES = list.length;
        for (var i in this.orbList) {
            this.scene.remove(this.orbList[i]);
        }
        for (var i in list) {
            var orb = list[i];
            var orbColor = new THREE.Color();
            orbColor.setHSL( Math.random(), 1.0, 0.5 );
            var orbSize = Math.sqrt(Math.sqrt(orb[3])) * 10;
            var sphere = new THREE.Mesh(new THREE.SphereGeometry(orbSize, 12, 12), new THREE.MeshLambertMaterial({ color: orbColor }));
            //sphere.rotation.y = -Math.PI / 2;
            sphere.position.x = orb[0];
            sphere.position.y = orb[1];
            sphere.position.z = orb[2];
            sphere.castShadow = sphere.receiveShadow = true;
            //sphere.geometry.radius = 500;//Math.sqrt(Math.sqrt(orb[3])) * 100//50;// * orb.sz;// 此值对显示大小无效
            
            this.scene.add(sphere);
            this.orbList[i] = sphere;//.push(sphere);
            this.isInited = true;
        }
    },
    updateOrbs: function(list) {
        if (list==undefined) {
            return false;
        }
        if (list.length != NUM_PARTICLES) {
            return this.initOrbs(list);
        }
        for (var i in list) {
            var orb = list[i];
            var sphere = this.orbList[i];
            //console.log(sphere);
            sphere.position.x = orb[0] * zoomBase;
            sphere.position.y = orb[1] * zoomBase;
            sphere.position.z = orb[2] * zoomBase;
            //sphere.geometry.radius = 50 * orb.sz;
        }
    }
};

var RecvMessage = function(data) {
    var cmd = data.data.cmd || undefined;
    switch (cmd) {
        case 'orbs':
            lesson1.updateOrbs(data.data.list);
            lesson1.dataList = data.data.list;
            if (enableCalc) {
                CalcUnit.stage = data.data.stage;
                CalcUnit.setOrbList(data.data.list);
                CalcUnit.key = data.data.key;
            }
            break;
        case 'taketask':
            //console.log(data);
            if (enableCalc) {
                CalcUnit.stage = data.data.stage;
                CalcUnit.setFeedList(data.data.feedlist);
                CalcUnit.consume();
            }
            break;
        case 'recvorb':
            //console.log(data);
            break;
        default:
            lesson1.updateOrbs(data.data.list);
            lesson1.dataList = data.data.list;
            //console.log("unknown cmd:");
            //console.log(data);
            break;
    }
}

// Animate the scene
function animate() {
    requestAnimationFrame(animate);
    render();
    update();
}

// Update controls and stats
function update() {
    lesson1.controls.update(lesson1.clock.getDelta());
    lesson1.stats.update();

    // 从服务器取数据，显示到屏幕
    //MyWebsocket.sceneMgr = lesson1;
    ++ticker;
    if ((ticker+1)%25 == 1) {
        MyWebsocket.doSend(sendVal);
    }
    //// smoothly move the particleLight
    //var timer = Date.now() * 0.000025;
    //particleLight.position.x = Math.sin(timer * 5) * 300;
    //particleLight.position.z = Math.cos(timer * 5) * 300;
}

// Render the scene
function render() {
    if (lesson1.renderer) {
        lesson1.renderer.render(lesson1.scene, lesson1.camera);
    }
}

// Initialize lesson on page load
function initializeLesson() {
    lesson1.init();
    MyWebsocket.receiveCallback = RecvMessage;//不能是lesson1.updateOrbs,函数复制有没有对象。

    if (wsUri == undefined || wsUri.length==0) {
        wsUri = $('#ws-addr').val();
        if (window.document.domain != undefined) {
            
            wsUri = 'ws://'+window.document.domain+':8082'+'/orbs';
            $('#ws-addr').val(wsUri);
            MyWebsocket.wsUri = wsUri;
        }
    }
    if (wsUri.length > 0) {
        MyWebsocket.initWebsocket();
    }

    $('#ws-addr').val(wsUri);
    $('#zoom_up').on('click', function() {
        zoomBase = zoomBase*zoomStep;
        $('#zoom').val(zoomBase);
    });
    $('#zoom_down').on('click', function() {
        zoomBase = zoomBase/zoomStep;
        $('#zoom').val(zoomBase);
    });
    $('#reConnect').on('click', function() {
        wsUri = $('#ws-addr').val();
        MyWebsocket.wsUri = wsUri;
        //alert(wsUri);
        MyWebsocket.initWebsocket();
    });
    $('#btnSend').on('click', function() {
        sendVal = $('#send-val').val();
    });
    $('#btnEnableCalc').on('click', function() {
        if (enableCalc) {
            enableCalc = false;
            $('#btnEnableCalc').html('do help calc');
        } else {
            enableCalc = true;
            CalcUnit.reloadOrbList();
            CalcUnit.reloadFeedList();
            $('#btnEnableCalc').html('stop help calc');
        }
    });
    
    animate();
}


if (window.addEventListener)
    window.addEventListener('load', initializeLesson, false);
else if (window.attachEvent)
    window.attachEvent('onload', initializeLesson);
else window.onload = initializeLesson;
