import { createRxDatabase, addRxPlugin } from 'rxdb';
import type { RxDatabase, RxCollection } from 'rxdb';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { RxDBQueryBuilderPlugin } from 'rxdb/plugins/query-builder';

// Enable query builder
addRxPlugin(RxDBQueryBuilderPlugin);

/* ------------------------------------------------------------------ */
/*  Schemas                                                            */
/* ------------------------------------------------------------------ */

const telemetrySchema = {
    title: 'telemetry schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        time: { type: 'string' },
        reqPerSec: { type: 'number' },
        errorPct: { type: 'number' },
        p50: { type: 'number' },
        p95: { type: 'number' },
        p99: { type: 'number' },
        ebpfSyscalls: {
            type: 'object',
            properties: {
                read: { type: 'number' },
                write: { type: 'number' },
                open: { type: 'number' },
                close: { type: 'number' },
                execve: { type: 'number' }
            }
        },
        timestamp: { type: 'number' }
    },
    required: ['id', 'time', 'timestamp']
};

const agentLogSchema = {
    title: 'agent log schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        timestamp: { type: 'number' },
        agent: { type: 'string' },
        severity: { type: 'string' },
        message: { type: 'string' },
        reasoning: { type: 'array', items: { type: 'string' } }
    },
    required: ['id', 'timestamp', 'agent', 'severity', 'message']
};

const walEventSchema = {
    title: 'wal event schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        type: { type: 'string' },
        updatedAt: { type: 'number' },
        deleted: { type: 'boolean' },
        payload: { type: 'object', additionalProperties: true }
    },
    required: ['id', 'type', 'updatedAt', 'deleted']
};

const agentGraphStateSchema = {
    title: 'agent graph state schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        label: { type: 'string' },
        model: { type: 'string' },
        color: { type: 'string' },
        status: { type: 'string' }
    },
    required: ['id', 'status']
};

const sandboxStateSchema = {
    title: 'sandbox state schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        name: { type: 'string' },
        status: { type: 'string' },
        cpuPercent: { type: 'number' },
        memoryPercent: { type: 'number' },
        memoryMb: { type: 'number' },
        memoryLimitMb: { type: 'number' },
        uptimeSeconds: { type: 'number' },
        logs: {
            type: 'array',
            items: {
                type: 'object',
                properties: {
                    time: { type: 'string' },
                    message: { type: 'string' }
                }
            }
        }
    },
    required: ['id', 'status']
};


const contextCacheSchema = {
    title: 'context cache schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        name: { type: 'string' },
        model: { type: 'string' },
        tokenCount: { type: 'number' },
        contextWindow: { type: 'number' },
        ttlSeconds: { type: 'number' },
        agentCount: { type: 'number' },
        createdAt: { type: 'number' }
    },
    required: ['id', 'name', 'model', 'tokenCount', 'contextWindow', 'ttlSeconds', 'agentCount', 'createdAt']
};

const taskQueueSchema = {
    title: 'task queue schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        title: { type: 'string' },
        agent: { type: 'string' },
        tier: { type: 'string' },
        priority: { type: 'string' },
        time: { type: 'string' },
        column: { type: 'string' },
        view_mode: { type: 'string' },
        extended_thinking: { type: 'boolean' },
        extra: {
            type: 'object',
            properties: {
                proposedChanges: { type: 'string' }
            }
        }
    },
    required: ['id', 'title', 'agent', 'tier', 'priority', 'time', 'column']
};

const chatMessageSchema = {
    title: 'chat message schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        role: { type: 'string' },
        content: { type: 'string' },
        agentName: { type: 'string' },
        timestamp: { type: 'number' },
        workspaceId: { type: 'string' },
        threadId: { type: 'string' },
        modelUsed: { type: 'string' },
        reasoning: { type: 'array', items: { type: 'string' } }
    },
    required: ['id', 'role', 'content', 'timestamp']
};

const chatThreadSchema = {
    title: 'chat thread schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        workspaceId: { type: 'string' },
        title: { type: 'string' },
        updatedAt: { type: 'number' },
        pinned: { type: 'boolean' }
    },
    required: ['id', 'workspaceId', 'title', 'updatedAt']
};

const educationCourseSchema = {
    title: 'education course schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        name: { type: 'string' },
        code: { type: 'string' },
        color: { type: 'string' }
    },
    required: ['id', 'name', 'code', 'color']
};

const educationAssignmentSchema = {
    title: 'education assignment schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        courseId: { type: 'string' },
        title: { type: 'string' },
        tag: { type: 'string' },
        date: { type: 'string' },
        status: { type: 'string' } // 'To Do', 'In Progress', 'Completed'
    },
    required: ['id', 'courseId', 'title', 'tag', 'date', 'status']
};

const agentIntentSchema = {
    title: 'agent intent schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        intentType: { type: 'string' },
        workspaceId: { type: 'string' },
        agentName: { type: 'string' },
        payload: { type: 'object', additionalProperties: true },
        timestamp: { type: 'number' },
        status: { type: 'string' } // 'pending', 'consumed', 'error'
    },
    required: ['id', 'intentType', 'timestamp', 'status']
};

const vlmFrameSchema = {
    title: 'vlm frame schema',
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        timestamp: { type: 'number' },
        base64Data: { type: 'string' }
    },
    required: ['id', 'timestamp', 'base64Data'],
    indexes: ['timestamp']
};

/* ------------------------------------------------------------------ */
/*  Database Types                                                     */
/* ------------------------------------------------------------------ */

export type TelemetryDocType = {
    id: string;
    time: string;
    reqPerSec?: number;
    errorPct?: number;
    p50?: number;
    p95?: number;
    p99?: number;
    ebpfSyscalls?: {
        read: number;
        write: number;
        open: number;
        close: number;
        execve: number;
    };
    timestamp: number;
};

export type AgentLogDocType = {
    id: string;
    timestamp: number;
    agent: string;
    severity: 'info' | 'warn' | 'error' | 'debug';
    message: string;
    reasoning?: string[];
};

export type WALEventDocType = {
    id: string;
    type: 'INSERT' | 'UPDATE' | 'DELETE' | 'UPSERT';
    updatedAt: number;
    deleted: boolean;
    payload: Record<string, any>;
};

export type AgentGraphStateDocType = {
    id: string;
    label?: string;
    model?: string;
    color?: string;
    status: 'idle' | 'running' | 'complete' | 'success' | 'error';
};

export type SandboxStateDocType = {
    id: string;
    name: string;
    status: 'running' | 'stopped' | 'error';
    cpuPercent: number;
    memoryPercent: number;
    memoryMb: number;
    memoryLimitMb: number;
    uptimeSeconds: number;
    logs: Array<{ time: string | Date; message: string }>;
};

export type ContextCacheDocType = {
    id: string;
    name: string;
    model: string;
    tokenCount: number;
    contextWindow: number;
    ttlSeconds: number;
    agentCount: number;
    createdAt: number;
};

export type TaskQueueDocType = {
    id: string;
    title: string;
    agent: string;
    tier: string;
    priority: string;
    time: string;
    column: string; // 'pending' | 'active' | 'completed'
    view_mode?: string;
    extended_thinking?: boolean;
    extra?: {
        proposedChanges?: string;
    };
};

export type ChatMessageDocType = {
    id: string;
    role: string;
    content: string;
    agentName?: string;
    timestamp: number;
    workspaceId?: string;
    threadId?: string;
    modelUsed?: string;
    reasoning?: string[];
};

export type ChatThreadDocType = {
    id: string;
    workspaceId: string;
    title: string;
    updatedAt: number;
    pinned?: boolean;
};

export type EducationCourseDocType = {
    id: string;
    name: string;
    code: string;
    color: string;
};

export type EducationAssignmentDocType = {
    id: string;
    courseId: string;
    title: string;
    tag: string;
    date: string;
    status: 'To Do' | 'In Progress' | 'Completed';
};

export type AgentIntentDocType = {
    id: string;
    intentType: string;
    workspaceId?: string;
    agentName?: string;
    payload: Record<string, any>;
    timestamp: number;
    status: 'pending' | 'consumed' | 'error';
};

export type VlmFrameDocType = {
    id: string;
    timestamp: number;
    base64Data: string;
};

export type AiofDatabaseCollections = {
    telemetry: RxCollection<TelemetryDocType>;
    agent_logs: RxCollection<AgentLogDocType>;
    wal_events: RxCollection<WALEventDocType>;
    agent_graph_state: RxCollection<AgentGraphStateDocType>;
    sandbox_state: RxCollection<SandboxStateDocType>;
    context_cache: RxCollection<ContextCacheDocType>;
    task_queue: RxCollection<TaskQueueDocType>;
    chat_messages: RxCollection<ChatMessageDocType>;
    chat_threads: RxCollection<ChatThreadDocType>;
    education_courses: RxCollection<EducationCourseDocType>;
    education_assignments: RxCollection<EducationAssignmentDocType>;
    agent_intents: RxCollection<AgentIntentDocType>;
    vlm_frames: RxCollection<VlmFrameDocType>;
};

export type AiofDatabase = RxDatabase<AiofDatabaseCollections>;

/* ------------------------------------------------------------------ */
/*  Initialization                                                     */
/* ------------------------------------------------------------------ */

import { useLayoutStore } from '../hooks/useLayoutStore';

import SHA256 from 'crypto-js/sha256';

if (!(window as any).__dbPromises) {
    (window as any).__dbPromises = {};
}
const storage = (window as any).__rxStorage || getRxStorageDexie();
(window as any).__rxStorage = storage;

export function getDatabase(projectId?: string): Promise<AiofDatabase> {
    const id = projectId || useLayoutStore.getState().activeProject;
    if (!id) throw new Error('projectId required for database (no active project)');
    const sanitizedId = id.replace(/[^a-zA-Z0-9]/g, '_');
    
    if (!(window as any).__dbPromises[sanitizedId]) {
        try {
            const dbName = 'aiofdb_v13_' + sanitizedId;
            const dbPromise = createRxDatabase<AiofDatabaseCollections>({
                name: dbName,
                storage: storage,
                hashFunction: async (input: any) => SHA256(input.toString()).toString(),
            }).then(async (db) => {
                await db.addCollections({
                    telemetry: { schema: telemetrySchema },
                    agent_logs: { schema: agentLogSchema },
                    wal_events: { schema: walEventSchema },
                    agent_graph_state: { schema: agentGraphStateSchema },
                    sandbox_state: { schema: sandboxStateSchema },
                    context_cache: { schema: contextCacheSchema },
                    task_queue: { schema: taskQueueSchema },
                    chat_messages: { schema: chatMessageSchema },
                    chat_threads: { schema: chatThreadSchema },
                    education_courses: { schema: educationCourseSchema },
                    education_assignments: { schema: educationAssignmentSchema },
                    agent_intents: { schema: agentIntentSchema },
                    vlm_frames: { schema: vlmFrameSchema }
                });
                return db;
            });
            (window as any).__dbPromises[sanitizedId] = dbPromise;
        } catch (err) {
            console.error('Failed to create RxDB:', err);
        }
    }
    return (window as any).__dbPromises[sanitizedId];
}
