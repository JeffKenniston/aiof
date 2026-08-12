import { use, useSyncExternalStore, useCallback } from 'react';
import type { RxQuery } from 'rxdb';

// A cache to hold Suspense promises mapping
const queryCache = new WeakMap<RxQuery<any, any>, { 
    promise: Promise<void> | null, 
    data: any | null, 
    resolved: boolean 
}>();

export function useRxQuery<T>(query: RxQuery<T, any>): T {
    // 1. Initialize cache state for this query
    let state = queryCache.get(query);
    if (!state) {
        state = { promise: null, data: null, resolved: false };
        
        state.promise = new Promise<void>((resolve) => {
            const sub = query.$.subscribe((result) => {
                state!.data = result;
                if (!state!.resolved) {
                    state!.resolved = true;
                    resolve();
                    sub.unsubscribe();
                }
            });
        });
        queryCache.set(query, state);
    }

    // 2. React 19 Suspense binding via use() semantics
    if (!state.resolved && state.promise) {
        use(state.promise); // Suspend component rendering until data is available
    }

    const subscribe = useCallback((onStoreChange: () => void) => {
        const subscription = query.$.subscribe((newData) => {
            state!.data = newData;
            onStoreChange();
        });
        return () => subscription.unsubscribe();
    }, [query, state]);

    // 3. React to ongoing RxDB mutations
    const data = useSyncExternalStore(
        subscribe,
        () => state!.data
    );

    return data;
}
