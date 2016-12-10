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

var lesson1 = {
    scene: null,
    camera: null,
    renderer: null,
    container: null,
    controls: null,
    clock: null,
    stats: null,
    orbList: {},// new Map();
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
        for (var i in list) {
            var orb = list[i];
            var sphere = new THREE.Mesh(new THREE.SphereGeometry(7, 12, 12), new THREE.MeshLambertMaterial({ color: orb.id*311331 }));
            //sphere.rotation.y = -Math.PI / 2;
            sphere.position.x = orb.y;
            sphere.position.y = orb.x;
            sphere.position.z = orb.z;
            sphere.castShadow = sphere.receiveShadow = true;
            sphere.geometry.radius = 50 * orb.size;
            
            this.scene.add(sphere);
            this.orbList[orb.id] = sphere;
            this.isInited = true;
        }
    },
    updateOrbs: function(list) {
        for (var i in list) {
            var orb = list[i];
            if (this.orbList.hasOwnProperty(orb.id)) {
                
                var sphere = this.orbList[orb.id];
                //console.log(sphere);
                if (orb.lifeStep!=1) {
                    // remove sphere
                    this.scene.remove(sphere);
                } else {
                    sphere.position.x = orb.y * zoomBase;
                    sphere.position.y = orb.x * zoomBase;
                    sphere.position.z = orb.z * zoomBase;
                    //sphere.geometry.radius = 50 * orb.size;
                }
            } else {
                //console.log('id='+orb.id+' not exist in orbList, will ');
                //add new sphere
                var sphere = new THREE.Mesh(new THREE.SphereGeometry(8, 12, 12), new THREE.MeshLambertMaterial({ color: orb.id*311331 }));
                //sphere.rotation.y = -Math.PI / 2;
                sphere.position.x = orb.y * zoomBase;
                sphere.position.y = orb.x * zoomBase;
                sphere.position.z = orb.z * zoomBase;
                sphere.castShadow = sphere.receiveShadow = true;
                //sphere.geometry.radius = 50 * orb.size;
                
                this.scene.add(sphere);
                this.orbList[orb.id] = sphere;
            }
        }
    }
};

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
    if ((ticker+1)%5 == 1) {
        MyWebsocket.doSend('k='+mcKey);
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
    MyWebsocket.sceneMgr = lesson1;
    MyWebsocket.initWebsocket();
    //MyWebsocket.doSend('k='+mcKey);
        $('#zoom_up').on('click', function() {
            zoomBase = zoomBase*zoomStep;
            $('#zoom').val(zoomBase);
        });
        $('#zoom_down').on('click', function() {
            zoomBase = zoomBase/zoomStep;
            $('#zoom').val(zoomBase);
        });
    
    animate();
}

if (window.addEventListener)
    window.addEventListener('load', initializeLesson, false);
else if (window.attachEvent)
    window.attachEvent('onload', initializeLesson);
else window.onload = initializeLesson;
