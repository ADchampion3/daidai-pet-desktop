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
