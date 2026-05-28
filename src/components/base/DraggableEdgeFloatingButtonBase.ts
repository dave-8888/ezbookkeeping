import { computed, onMounted, onUnmounted, ref, type CSSProperties } from 'vue';

export type FloatingEdge = 'left' | 'right' | 'top' | 'bottom';

export interface DraggableEdgeFloatingButtonOptions {
    readonly storageKey: string;
    readonly buttonSize: number;
    readonly marginX: number;
    readonly marginTop: number;
    readonly marginBottom: number;
    readonly initialEdge?: FloatingEdge;
    readonly dragThreshold?: number;
    readonly onDragStart?: () => void;
}

interface SavedFloatingButtonPosition {
    readonly edge?: FloatingEdge;
    readonly alongRatio?: number;
    readonly topRatio?: number;
}

interface DragState {
    readonly pointerId: number;
    readonly startX: number;
    readonly startY: number;
    readonly startLeft: number;
    readonly startTop: number;
    moved: boolean;
}

export function useDraggableEdgeFloatingButton(options: DraggableEdgeFloatingButtonOptions) {
    const edge = ref<FloatingEdge>(options.initialEdge || 'right');
    const left = ref<number>(getDefaultLeft());
    const top = ref<number>(getDefaultTop());
    const dragging = ref<boolean>(false);
    const suppressNextClick = ref<boolean>(false);

    let dragState: DragState | null = null;

    const buttonStyle = computed<CSSProperties>(() => {
        return {
            left: `${left.value}px`,
            top: `${top.value}px`
        };
    });

    function getViewportWidth(): number {
        return window.innerWidth || document.documentElement.clientWidth || 0;
    }

    function getViewportHeight(): number {
        return window.innerHeight || document.documentElement.clientHeight || 0;
    }

    function getMinimumTop(): number {
        return options.marginTop;
    }

    function getMaximumLeft(): number {
        return Math.max(options.marginX, getViewportWidth() - options.marginX - options.buttonSize);
    }

    function getMaximumTop(): number {
        return Math.max(getMinimumTop(), getViewportHeight() - options.marginBottom - options.buttonSize);
    }

    function clampLeft(value: number): number {
        return Math.min(Math.max(value, options.marginX), getMaximumLeft());
    }

    function clampTop(value: number): number {
        return Math.min(Math.max(value, getMinimumTop()), getMaximumTop());
    }

    function getDefaultLeft(): number {
        if (typeof window === 'undefined') {
            return options.marginX;
        }

        return getMaximumLeft();
    }

    function getDefaultTop(): number {
        if (typeof window === 'undefined') {
            return options.marginTop;
        }

        return clampTop(getViewportHeight() - options.marginBottom - options.buttonSize);
    }

    function getTopRatio(): number {
        const minTop = getMinimumTop();
        const range = getMaximumTop() - minTop;

        if (range <= 0) {
            return 0;
        }

        return (top.value - minTop) / range;
    }

    function getLeftRatio(): number {
        const range = getMaximumLeft() - options.marginX;

        if (range <= 0) {
            return 0;
        }

        return (left.value - options.marginX) / range;
    }

    function setTopByRatio(ratio: number): void {
        const minTop = getMinimumTop();
        const range = getMaximumTop() - minTop;

        top.value = clampTop(minTop + (Math.min(Math.max(ratio, 0), 1) * range));
    }

    function setLeftByRatio(ratio: number): void {
        const range = getMaximumLeft() - options.marginX;

        left.value = clampLeft(options.marginX + (Math.min(Math.max(ratio, 0), 1) * range));
    }

    function getAlongRatio(): number {
        if (edge.value === 'top' || edge.value === 'bottom') {
            return getLeftRatio();
        }

        return getTopRatio();
    }

    function setPositionOnEdge(edgeValue: FloatingEdge, alongRatio: number): void {
        edge.value = edgeValue;

        if (edgeValue === 'left') {
            left.value = options.marginX;
            setTopByRatio(alongRatio);
        } else if (edgeValue === 'right') {
            left.value = getMaximumLeft();
            setTopByRatio(alongRatio);
        } else if (edgeValue === 'top') {
            top.value = getMinimumTop();
            setLeftByRatio(alongRatio);
        } else {
            top.value = getMaximumTop();
            setLeftByRatio(alongRatio);
        }
    }

    function getNearestEdge(): FloatingEdge {
        const distances: Record<FloatingEdge, number> = {
            left: left.value,
            right: getViewportWidth() - left.value - options.buttonSize,
            top: top.value,
            bottom: getViewportHeight() - top.value - options.buttonSize
        };

        let nearestEdge: FloatingEdge = 'left';

        for (const currentEdge of Object.keys(distances) as FloatingEdge[]) {
            if (distances[currentEdge] < distances[nearestEdge]) {
                nearestEdge = currentEdge;
            }
        }

        return nearestEdge;
    }

    function snapToNearestEdge(): void {
        const nearestEdge = getNearestEdge();
        const alongRatio = nearestEdge === 'top' || nearestEdge === 'bottom' ? getLeftRatio() : getTopRatio();

        setPositionOnEdge(nearestEdge, alongRatio);
    }

    function restorePosition(): void {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            const savedPosition = localStorage.getItem(options.storageKey);

            if (!savedPosition) {
                left.value = getDefaultLeft();
                top.value = getDefaultTop();
                return;
            }

            const position = JSON.parse(savedPosition) as SavedFloatingButtonPosition;
            const savedEdge = position.edge === 'left' || position.edge === 'right' || position.edge === 'top' || position.edge === 'bottom' ? position.edge : edge.value;
            const savedRatio = typeof position.alongRatio === 'number' ? position.alongRatio : (typeof position.topRatio === 'number' ? position.topRatio : getAlongRatio());

            setPositionOnEdge(savedEdge, savedRatio);
        } catch {
            left.value = getDefaultLeft();
            top.value = getDefaultTop();
        }
    }

    function savePosition(): void {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            localStorage.setItem(options.storageKey, JSON.stringify({
                edge: edge.value,
                alongRatio: getAlongRatio()
            }));
        } catch {
            // Ignore storage failures; drag position can still work for the current session.
        }
    }

    function onPointerMove(event: PointerEvent): void {
        if (!dragState || event.pointerId !== dragState.pointerId) {
            return;
        }

        const movedDistance = Math.hypot(event.clientX - dragState.startX, event.clientY - dragState.startY);

        if (!dragState.moved && movedDistance < (options.dragThreshold || 6)) {
            return;
        }

        if (!dragState.moved) {
            dragState.moved = true;
            dragging.value = true;
            options.onDragStart?.();
        }

        left.value = clampLeft(dragState.startLeft + event.clientX - dragState.startX);
        top.value = clampTop(dragState.startTop + event.clientY - dragState.startY);
        event.preventDefault();
    }

    function endDrag(event: PointerEvent): void {
        if (!dragState || event.pointerId !== dragState.pointerId) {
            return;
        }

        const hasMoved = dragState.moved;

        dragState = null;
        dragging.value = false;

        document.removeEventListener('pointermove', onPointerMove);
        document.removeEventListener('pointerup', endDrag);
        document.removeEventListener('pointercancel', endDrag);

        if (hasMoved) {
            snapToNearestEdge();
            suppressNextClick.value = true;
            savePosition();

            window.setTimeout(() => {
                suppressNextClick.value = false;
            }, 150);
        }
    }

    function onPointerDown(event: PointerEvent): void {
        if (event.button !== 0) {
            return;
        }

        dragState = {
            pointerId: event.pointerId,
            startX: event.clientX,
            startY: event.clientY,
            startLeft: left.value,
            startTop: top.value,
            moved: false
        };

        document.addEventListener('pointermove', onPointerMove);
        document.addEventListener('pointerup', endDrag);
        document.addEventListener('pointercancel', endDrag);
    }

    function consumeSuppressClick(): boolean {
        if (!suppressNextClick.value) {
            return false;
        }

        suppressNextClick.value = false;
        return true;
    }

    function onResize(): void {
        const currentEdge = edge.value;
        const ratio = getAlongRatio();

        setPositionOnEdge(currentEdge, ratio);
    }

    onMounted(() => {
        restorePosition();
        window.addEventListener('resize', onResize);
    });

    onUnmounted(() => {
        window.removeEventListener('resize', onResize);
        document.removeEventListener('pointermove', onPointerMove);
        document.removeEventListener('pointerup', endDrag);
        document.removeEventListener('pointercancel', endDrag);
    });

    return {
        edge,
        left,
        top,
        dragging,
        buttonStyle,
        consumeSuppressClick,
        onPointerDown
    };
}
