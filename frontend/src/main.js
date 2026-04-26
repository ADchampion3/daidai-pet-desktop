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
let dragEnabled = true;
let runtimeListenersReady = false;
let startupPetSettingsSynced = false;
let startupDragSettingsSynced = false;

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
    pet.classList.remove('stand-l', 'stand-r', 'walk-l', 'walk-r');
    pet.classList.add(spriteKey);
}

function setDraggingState(active) {
    isDragging = dragEnabled && active;
    pet.style.cursor = dragEnabled ? (isDragging ? 'grabbing' : 'move') : 'default';
}

function setDragEnabled(enabled) {
    dragEnabled = Boolean(enabled);
    pet.classList.toggle('drag-disabled', !dragEnabled);
    if (!dragEnabled) {
        setDraggingState(false);
        return;
    }
    pet.style.cursor = 'move';
}

function startDrag() {
    if (!dragEnabled) {
        return;
    }

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
    if (!dragEnabled) {
        return;
    }

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
    if (!dragEnabled) {
        return;
    }

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

        window.runtime.EventsOn('updateDragState', (data) => {
            setDragEnabled(data.enabled);
        });

        window.runtime.EventsOn('updatePetSettings', applyPetSettings);
        runtimeListenersReady = true;
    }

    if (!window.go?.main?.App?.GetPetSettings || !window.go?.main?.App?.GetDragEnabled) {
        window.setTimeout(setupRuntimeListener, 100);
        return;
    }

    if (!startupPetSettingsSynced) {
        window.go.main.App.GetPetSettings()
            .then((data) => {
                applyPetSettings(data);
                startupPetSettingsSynced = true;
            })
            .catch(() => {
                window.setTimeout(setupRuntimeListener, 100);
            });
    }

    if (startupDragSettingsSynced) {
        return;
    }

    window.go.main.App.GetDragEnabled()
        .then((enabled) => {
            setDragEnabled(enabled);
            startupDragSettingsSynced = true;
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
