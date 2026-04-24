import './style.css';
import standRightUrl from './assets/stand_r.png';
import standLeftUrl from './assets/stand_l.png';
import walkRightUrl from './assets/walk_r.png';
import walkLeftUrl from './assets/walk_l.png';

const pet = document.getElementById('pet');
const root = document.documentElement;

const spriteUrls = {
    'stand-l': standLeftUrl,
    'stand-r': standRightUrl,
    'walk-l': walkLeftUrl,
    'walk-r': walkRightUrl,
};

let currentDir = 'r';
let currentState = 'stand';
let isDragging = false;
let runtimeListenersReady = false;
let startupPetSettingsSynced = false;

function applyPetSettings(data) {
    if (typeof data?.width === 'number' && Number.isFinite(data.width)) {
        root.style.setProperty('--pet-width', `${data.width}px`);
    }

    if (typeof data?.height === 'number' && Number.isFinite(data.height)) {
        root.style.setProperty('--pet-height', `${data.height}px`);
    }
}

function updateSprite() {
    const spriteKey = `${currentState}-${currentDir}`;
    root.style.setProperty('--pet-sprite', `url("${spriteUrls[spriteKey]}")`);
    pet.className = spriteKey;
}

function setDraggingState(active) {
    isDragging = active;
    pet.style.cursor = active ? 'grabbing' : 'move';
}

function startDrag() {
    setDraggingState(true);

    if (window.go?.main?.App) {
        window.go.main.App.SetDragStart();
    }
}

function endDrag() {
    if (!isDragging) {
        return;
    }

    setDraggingState(false);

    if (window.go?.main?.App) {
        window.go.main.App.SetDragEnd();
    }
}

function onMouseDown(event) {
    if (event.button !== 0) {
        return;
    }

    event.preventDefault();
    startDrag();
}

function onMouseMove(event) {
    if (!isDragging) {
        return;
    }

    event.preventDefault();
}

function onTouchStart(event) {
    if (event.touches.length !== 1) {
        return;
    }

    event.preventDefault();
    startDrag();
}

function onTouchMove(event) {
    if (!isDragging || event.touches.length !== 1) {
        return;
    }

    event.preventDefault();
}

function setupRuntimeListener() {
    if (!window.runtime?.EventsOn) {
        window.setTimeout(setupRuntimeListener, 100);
        return;
    }

    if (!runtimeListenersReady) {
        window.runtime.EventsOn('updatePositionState', (data) => {
            currentState = data.state;
            currentDir = data.dir;
            updateSprite();
        });

        window.runtime.EventsOn('dragEnded', () => {
            setDraggingState(false);
        });

        window.runtime.EventsOn('updatePetSettings', applyPetSettings);
        runtimeListenersReady = true;
    }

    if (!window.go?.main?.App?.GetPetSettings) {
        window.setTimeout(setupRuntimeListener, 100);
        return;
    }

    if (startupPetSettingsSynced) {
        return;
    }

    window.go.main.App.GetPetSettings()
        .then((data) => {
            applyPetSettings(data);
            startupPetSettingsSynced = true;
        })
        .catch(() => {
            window.setTimeout(setupRuntimeListener, 100);
        });
}

window.showPet = () => pet.classList.remove('hidden');
window.hidePet = () => pet.classList.add('hidden');

pet.addEventListener('mousedown', onMouseDown);
document.addEventListener('mousemove', onMouseMove);
document.addEventListener('mouseup', endDrag);
pet.addEventListener('touchstart', onTouchStart, { passive: false });
document.addEventListener('touchmove', onTouchMove, { passive: false });
document.addEventListener('touchend', endDrag);

updateSprite();
setupRuntimeListener();
