import { useState, use, Suspense, useMemo, useEffect } from 'react';

import { BookOpen, Calendar, LayoutDashboard, Plus, MoreVertical, Search, Bell, BrainCircuit, ChevronLeft, ChevronRight, RotateCw, Edit2, Trash2 } from 'lucide-react';
import { getDatabase, type EducationCourseDocType, type EducationAssignmentDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';

function EducationWorkstationContent() {
  const db = use(getDatabase());
  
  const coursesQuery = useMemo(() => db.education_courses.find(), [db]);
  const courses = useRxQuery<EducationCourseDocType[]>(coursesQuery as any);

  const assignmentsQuery = useMemo(() => db.education_assignments.find(), [db]);
  const assignments = useRxQuery<EducationAssignmentDocType[]>(assignmentsQuery as any);

  const [activeTab, setActiveTab] = useState<'calendar' | 'assignments' | 'agent'>('calendar');
  const [activeCourse, setActiveCourse] = useState<string | null>(courses && courses.length > 0 ? courses[0].id : null);
  
  // Create UI states
  const [isAddingCourse, setIsAddingCourse] = useState(false);
  const [newCourseName, setNewCourseName] = useState('');
  const [newCourseCode, setNewCourseCode] = useState('');

  const handleCreateCourse = async () => {
    if (!newCourseName) return;
    const newId = crypto.randomUUID();
    await db.education_courses.insert({
      id: newId,
      name: newCourseName,
      code: newCourseCode,
      color: 'bg-blue-500'
    });
    setNewCourseName('');
    setNewCourseCode('');
    setIsAddingCourse(false);
    if (!activeCourse) setActiveCourse(newId);
  };

  const activeCourseName = courses?.find(c => c.id === activeCourse)?.name || 'Select a Course';
  
  // Filter assignments by active course
  const activeAssignments = assignments ? assignments.filter(a => a.courseId === activeCourse) : [];

  const globalIntentsQuery = useMemo(() => db.agent_intents.find({
    selector: {
      intentType: { $in: ['CREATE_ASSIGNMENT', 'UPDATE_ASSIGNMENT_STATUS', 'DELETE_ASSIGNMENT', 'CREATE_COURSE'] },
      status: 'pending'
    }
  }), [db]);
  const globalIntents = useRxQuery<any[]>(globalIntentsQuery as any);

  useEffect(() => {
    if (!globalIntents || globalIntents.length === 0) return;
    globalIntents.forEach(async (intent) => {
      try {
        if (intent.intentType === 'CREATE_ASSIGNMENT') {
          const newId = crypto.randomUUID();
          await db.education_assignments.insert({
            id: newId,
            courseId: activeCourse || 'default',
            title: intent.payload?.title || 'New Task',
            tag: intent.payload?.tag || 'General',
            date: intent.payload?.date || new Date().toISOString().split('T')[0],
            status: intent.payload?.status || 'To Do'
          });
          await intent.patch({ status: 'consumed' });
        } else if (intent.intentType === 'UPDATE_ASSIGNMENT_STATUS') {
          const doc = await db.education_assignments.findOne(intent.payload?.assignmentId).exec();
          if (doc) await doc.patch({ status: intent.payload?.status });
          await intent.patch({ status: 'consumed' });
        } else if (intent.intentType === 'DELETE_ASSIGNMENT') {
          const doc = await db.education_assignments.findOne(intent.payload?.assignmentId).exec();
          if (doc) await doc.remove();
          await intent.patch({ status: 'consumed' });
        } else if (intent.intentType === 'CREATE_COURSE') {
          const newId = crypto.randomUUID();
          await db.education_courses.insert({
            id: newId,
            name: intent.payload?.name || 'New Course',
            code: intent.payload?.code || 'XXX 101',
            color: 'bg-blue-500'
          });
          await intent.patch({ status: 'consumed' });
        }
      } catch (err) {
        console.error('Failed to process intent', err);
        await intent.patch({ status: 'error' });
      }
    });
  }, [globalIntents, db, activeCourse]);

  return (
    <div className="flex h-full bg-neutral-950 text-neutral-50 overflow-hidden">
      {/* Sidebar: Course Manager */}
      <div className="w-64 bg-neutral-900 border-r border-white/5 flex flex-col">
        <div className="p-4 border-b border-white/5 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-accent-blue" />
            <h2 className="font-semibold text-sm tracking-wide">COURSES</h2>
          </div>
          <button 
            onClick={() => setIsAddingCourse(!isAddingCourse)}
            className="p-1 hover:bg-white/10 rounded-md transition-colors"
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>
        
        {isAddingCourse && (
          <div className="p-3 bg-neutral-800 border-b border-white/5 flex flex-col gap-2">
            <input 
              autoFocus
              className="px-2 py-1 bg-black/40 border border-white/10 rounded text-sm text-white" 
              placeholder="Course Name..." 
              value={newCourseName}
              onChange={(e) => setNewCourseName(e.target.value)}
            />
            <input 
              className="px-2 py-1 bg-black/40 border border-white/10 rounded text-sm text-white" 
              placeholder="Course Code..." 
              value={newCourseCode}
              onChange={(e) => setNewCourseCode(e.target.value)}
            />
            <div className="flex justify-end gap-2 mt-1">
              <button className="text-xs text-neutral-400 hover:text-white" onClick={() => setIsAddingCourse(false)}>Cancel</button>
              <button className="text-xs bg-accent-blue text-white px-2 py-1 rounded" onClick={handleCreateCourse}>Save</button>
            </div>
          </div>
        )}

        <div className="p-3 flex-1 overflow-y-auto space-y-1">
          {courses?.map(course => (
            <button 
              key={course.id}
              onClick={() => setActiveCourse(course.id)}
              className={`w-full flex items-center justify-between p-2.5 rounded-lg transition-colors ${activeCourse === course.id ? 'bg-white/10' : 'hover:bg-white/5'}`}
            >
              <div className="flex items-center gap-3 truncate">
                <div className={`w-2.5 h-2.5 rounded-full ${course.color}`} />
                <div className="text-left truncate">
                  <div className={`text-sm font-medium ${activeCourse === course.id ? 'text-white' : 'text-neutral-300'}`}>{course.name}</div>
                  <div className="text-xs text-neutral-500">{course.code}</div>
                </div>
              </div>
              <MoreVertical className="w-4 h-4 text-neutral-500" />
            </button>
          ))}
          {(!courses || courses.length === 0) && !isAddingCourse && (
            <div className="text-center p-4 text-xs text-neutral-500">No courses yet</div>
          )}
        </div>
        
        <div className="p-4 border-t border-white/5 bg-neutral-900/50">
          <div className="flex items-center gap-2 text-sm text-neutral-400">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            Active Semester
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="h-14 border-b border-white/5 flex items-center justify-between px-6 bg-neutral-900/30 backdrop-blur-sm">
          <div className="flex items-center gap-6">
            <h1 className="font-bold text-lg">{activeCourseName}</h1>
            
            <div className="flex items-center bg-black/40 rounded-lg p-1 border border-white/5">
              <button 
                onClick={() => setActiveTab('calendar')}
                className={`px-3 py-1 text-sm rounded-md flex items-center gap-2 transition-all ${activeTab === 'calendar' ? 'bg-white/10 text-white' : 'text-neutral-400 hover:text-white'}`}
              >
                <Calendar className="w-4 h-4" /> Calendar
              </button>
              <button 
                onClick={() => setActiveTab('assignments')}
                className={`px-3 py-1 text-sm rounded-md flex items-center gap-2 transition-all ${activeTab === 'assignments' ? 'bg-white/10 text-white' : 'text-neutral-400 hover:text-white'}`}
              >
                <LayoutDashboard className="w-4 h-4" /> Assignments
              </button>
              <button 
                onClick={() => setActiveTab('agent')}
                className={`px-3 py-1 text-sm rounded-md flex items-center gap-2 transition-all ${activeTab === 'agent' ? 'bg-accent-blue/20 text-accent-blue' : 'text-neutral-400 hover:text-accent-blue'}`}
              >
                <BrainCircuit className="w-4 h-4" /> Student Agent
              </button>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            <button className="p-2 text-neutral-400 hover:text-white"><Search className="w-4 h-4" /></button>
            <button className="p-2 text-neutral-400 hover:text-white relative">
              <Bell className="w-4 h-4" />
              <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-accent-orange rounded-full" />
            </button>
          </div>
        </header>

        {/* Dynamic View Content */}
        <main className="flex-1 overflow-hidden relative">
          {activeCourse ? (
             <>
              {activeTab === 'calendar' && <CalendarView assignments={activeAssignments} />}
              {activeTab === 'assignments' && <KanbanView assignments={activeAssignments} courseId={activeCourse} />}
              {activeTab === 'agent' && <VirtualBrowserView db={db} />}
             </>
          ) : (
            <div className="w-full h-full flex items-center justify-center text-neutral-500">
               Please create or select a course.
            </div>
          )}
        </main>
      </div>
    </div>
  );
}

function CalendarView({ assignments }: { assignments: EducationAssignmentDocType[] }) {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [viewMode, setViewMode] = useState<'month' | 'week'>('month');

  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();
  const today = new Date();

  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const firstDay = new Date(year, month, 1).getDay();
  
  const gridCells = [];
  
  if (viewMode === 'month') {
    for (let i = 0; i < firstDay; i++) gridCells.push({ day: null, date: null });
    for (let i = 1; i <= daysInMonth; i++) gridCells.push({ day: i, date: new Date(year, month, i) });
    while (gridCells.length % 7 !== 0 || gridCells.length < 35) gridCells.push({ day: null, date: null });
  } else {
    const currentDayOfWeek = currentDate.getDay();
    const startOfWeek = new Date(year, month, currentDate.getDate() - currentDayOfWeek);
    for (let i = 0; i < 7; i++) {
      const d = new Date(startOfWeek);
      d.setDate(startOfWeek.getDate() + i);
      gridCells.push({ day: d.getDate(), date: d });
    }
  }
  
  const monthName = currentDate.toLocaleString('default', { month: 'long' });

  const handleNext = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(year, month + 1, 1));
    } else {
      setCurrentDate(new Date(year, month, currentDate.getDate() + 7));
    }
  };

  const handlePrev = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(year, month - 1, 1));
    } else {
      setCurrentDate(new Date(year, month, currentDate.getDate() - 7));
    }
  };

  return (
    <div className="h-full p-6 flex flex-col">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <h3 className="text-xl font-medium w-48">{monthName} {year}</h3>
          <div className="flex gap-1 bg-white/5 rounded-lg p-1 border border-white/5">
            <button onClick={handlePrev} className="p-1 hover:bg-white/10 rounded-md text-neutral-400 hover:text-white transition-colors"><ChevronLeft className="w-5 h-5" /></button>
            <button onClick={() => setCurrentDate(new Date())} className="px-3 py-1 text-sm text-neutral-400 hover:text-white transition-colors">Today</button>
            <button onClick={handleNext} className="p-1 hover:bg-white/10 rounded-md text-neutral-400 hover:text-white transition-colors"><ChevronRight className="w-5 h-5" /></button>
          </div>
        </div>
        <div className="flex gap-1 bg-white/5 rounded-lg p-1 border border-white/5">
          <button onClick={() => setViewMode('month')} className={`px-4 py-1.5 rounded-md text-sm transition-all ${viewMode === 'month' ? 'bg-white/10 text-white shadow-sm' : 'text-neutral-400 hover:text-white'}`}>Month</button>
          <button onClick={() => setViewMode('week')} className={`px-4 py-1.5 rounded-md text-sm transition-all ${viewMode === 'week' ? 'bg-white/10 text-white shadow-sm' : 'text-neutral-400 hover:text-white'}`}>Week</button>
        </div>
      </div>
      <div className={`flex-1 grid grid-cols-7 ${viewMode === 'month' ? 'grid-rows-5' : 'grid-rows-1'} gap-px bg-white/5 border border-white/5 rounded-xl overflow-hidden shadow-2xl`}>
        {gridCells.map((cell, i) => {
          const isToday = cell.date && cell.date.getDate() === today.getDate() && cell.date.getMonth() === today.getMonth() && cell.date.getFullYear() === today.getFullYear();
          
          const dayAssignments = cell.date ? assignments.filter(a => {
             const d = new Date(a.date);
             return !isNaN(d.getTime()) && d.getDate() === cell.date?.getDate() && d.getMonth() === cell.date?.getMonth() && d.getFullYear() === cell.date?.getFullYear();
          }) : [];
          
          return (
            <div key={i} className={`bg-neutral-950 p-2 transition-colors ${dayAssignments.length > 0 ? 'bg-neutral-900/80' : ''} ${isToday ? 'ring-1 ring-inset ring-accent-blue/50 bg-accent-blue/[0.02]' : ''}`}>
              <span className={`text-xs ${isToday ? 'bg-accent-blue text-white px-2 py-0.5 rounded-full font-bold shadow-[0_0_10px_rgba(59,130,246,0.5)]' : dayAssignments.length > 0 ? 'text-white font-bold' : 'text-neutral-600'}`}>
                {cell.day || ''}
              </span>
              <div className="mt-2 space-y-1 overflow-y-auto max-h-[80%] custom-scrollbar">
                {dayAssignments.map(a => (
                  <div key={a.id} className="text-[11px] bg-blue-500/10 text-blue-300 p-1.5 rounded border border-blue-500/20 truncate hover:bg-blue-500/20 hover:border-blue-500/40 transition-colors cursor-pointer" title={a.title}>
                    {a.title}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function KanbanView({ assignments, courseId }: { assignments: EducationAssignmentDocType[], courseId: string }) {
  const db = use(getDatabase());
  const [isAddingTask, setIsAddingTask] = useState<string | null>(null);
  const [newTaskTitle, setNewTaskTitle] = useState('');
  const [newTaskTag, setNewTaskTag] = useState('');
  const [newTaskDate, setNewTaskDate] = useState('');
  const [contextMenu, setContextMenu] = useState<{ x: number, y: number, assignmentId: string } | null>(null);

  useEffect(() => {
    const close = () => setContextMenu(null);
    window.addEventListener('click', close);
    return () => window.removeEventListener('click', close);
  }, []);

  const handleCreateAssignment = async (status: string) => {
    if (!newTaskTitle) return;
    await db.education_assignments.insert({
      id: crypto.randomUUID(),
      courseId,
      title: newTaskTitle,
      tag: newTaskTag || 'Task',
      date: newTaskDate || new Date().toISOString().split('T')[0],
      status: status as any
    });
    setNewTaskTitle('');
    setNewTaskTag('');
    setNewTaskDate('');
    setIsAddingTask(null);
  };

  const handleDrop = async (e: React.DragEvent, status: string) => {
    e.preventDefault();
    const id = e.dataTransfer.getData('text/plain');
    const assignment = assignments.find(a => a.id === id);
    if (assignment && assignment.status !== status) {
      const doc = await db.education_assignments.findOne(id).exec();
      if (doc) await doc.patch({ status: status as any });
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const columns = ['To Do', 'In Progress', 'Completed'] as const;

  return (
    <div className="h-full p-6 flex gap-6 overflow-x-auto relative">
      {contextMenu && (
        <div 
          className="fixed z-50 bg-neutral-900 border border-white/10 rounded-lg shadow-2xl py-1 w-36 backdrop-blur-xl"
          style={{ top: contextMenu.y, left: contextMenu.x }}
        >
          <button 
            className="w-full text-left px-3 py-2 text-sm hover:bg-white/10 text-neutral-200 flex items-center gap-2 transition-colors"
            onClick={() => {/* Edit Placeholder */}}
          >
            <Edit2 className="w-3.5 h-3.5" /> Edit
          </button>
          <button 
            className="w-full text-left px-3 py-2 text-sm hover:bg-red-500/20 text-red-400 flex items-center gap-2 transition-colors"
            onClick={async () => {
              const doc = await db.education_assignments.findOne(contextMenu.assignmentId).exec();
              if (doc) await doc.remove();
            }}
          >
            <Trash2 className="w-3.5 h-3.5" /> Delete
          </button>
        </div>
      )}

      {columns.map((col) => {
        const colAssignments = assignments.filter(a => a.status === col);
        return (
          <div 
            key={col} 
            onDrop={(e) => handleDrop(e, col)}
            onDragOver={handleDragOver}
            className="flex-1 min-w-[300px] flex flex-col bg-white/[0.03] backdrop-blur-3xl rounded-2xl border border-white/5 shadow-xl overflow-hidden transition-colors"
          >
            <div className="p-4 border-b border-white/5 font-medium flex justify-between items-center bg-white/[0.02]">
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${col === 'Completed' ? 'bg-emerald-500' : col === 'In Progress' ? 'bg-accent-blue' : 'bg-neutral-500'}`} />
                {col}
                <span className="text-xs bg-black/50 px-2 py-1 rounded-full text-neutral-400 border border-white/5">
                  {colAssignments.length}
                </span>
              </div>
              <button 
                onClick={() => setIsAddingTask(col)}
                className="p-1 hover:bg-white/10 rounded-md text-neutral-400 hover:text-white transition-colors"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            
            <div className="p-4 flex-1 space-y-3 overflow-y-auto custom-scrollbar">
              {isAddingTask === col && (
                <div className="p-3 rounded-xl border border-accent-blue/30 bg-accent-blue/5 flex flex-col gap-2 shadow-lg">
                  <input 
                    autoFocus
                    className="w-full px-2 py-1.5 bg-black/40 border border-white/10 rounded-lg text-sm text-white outline-none focus:border-accent-blue/50" 
                    placeholder="Title..." 
                    value={newTaskTitle}
                    onChange={(e) => setNewTaskTitle(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <input 
                      className="w-1/2 px-2 py-1.5 bg-black/40 border border-white/10 rounded-lg text-sm text-white outline-none focus:border-accent-blue/50" 
                      placeholder="Tag (e.g. Reading)" 
                      value={newTaskTag}
                      onChange={(e) => setNewTaskTag(e.target.value)}
                    />
                    <input 
                      type="date"
                      className="w-1/2 px-2 py-1.5 bg-black/40 border border-white/10 rounded-lg text-sm text-white outline-none focus:border-accent-blue/50" 
                      value={newTaskDate}
                      onChange={(e) => setNewTaskDate(e.target.value)}
                    />
                  </div>
                  <div className="flex justify-end gap-2 mt-2">
                    <button className="text-xs text-neutral-400 hover:text-white px-2 py-1" onClick={() => setIsAddingTask(null)}>Cancel</button>
                    <button className="text-xs bg-accent-blue hover:bg-accent-blue/80 text-white px-3 py-1.5 rounded-lg font-medium transition-colors" onClick={() => handleCreateAssignment(col)}>Save</button>
                  </div>
                </div>
              )}
              
              {colAssignments.map(assignment => (
                <KanbanCard 
                  key={assignment.id} 
                  assignment={assignment}
                  active={col === 'In Progress'} 
                  onContextMenu={(e) => {
                    e.preventDefault();
                    setContextMenu({ x: e.clientX, y: e.clientY, assignmentId: assignment.id });
                  }}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function KanbanCard({ assignment, active = false, onContextMenu }: { assignment: EducationAssignmentDocType, active?: boolean, onContextMenu: (e: React.MouseEvent) => void }) {
  return (
    <div 
      draggable
      onDragStart={(e) => e.dataTransfer.setData('text/plain', assignment.id)}
      onContextMenu={onContextMenu}
      className={`p-4 rounded-xl border ${active ? 'bg-accent-blue/[0.05] border-accent-blue/30 shadow-[0_4px_20px_rgba(59,130,246,0.1)]' : 'bg-neutral-900 border-white/5 shadow-md'} hover:border-white/20 transition-all cursor-grab active:cursor-grabbing group`}
    >
      <div className="text-xs text-neutral-500 mb-2 flex justify-between items-center">
        <span className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wider ${active ? 'bg-accent-blue/20 text-accent-blue' : 'bg-white/10 text-neutral-300'}`}>{assignment.tag}</span>
        <span className="flex items-center gap-1 opacity-70 group-hover:opacity-100 transition-opacity"><Calendar className="w-3 h-3" /> {assignment.date}</span>
      </div>
      <h4 className="text-sm font-medium text-neutral-200 group-hover:text-white transition-colors">{assignment.title}</h4>
    </div>
  );
}

function VirtualBrowserView({ db }: { db: any }) {
  const [url, setUrl] = useState('https://en.wikipedia.org/wiki/Main_Page');
  const [inputUrl, setInputUrl] = useState(url);

  const intentsQuery = useMemo(() => db.agent_intents.find({
    selector: {
      intentType: 'NAVIGATE_URL',
      status: 'pending'
    }
  }), [db]);

  const pendingIntents = useRxQuery<any[]>(intentsQuery as any);

  useEffect(() => {
    if (pendingIntents && pendingIntents.length > 0) {
      const intent = pendingIntents[0];
      const newUrl = intent.payload?.url;
      if (newUrl) {
        setUrl(newUrl);
        setInputUrl(newUrl);
        intent.patch({ status: 'consumed' }).catch(console.error);
      }
    }
  }, [pendingIntents]);

  const framesQuery = useMemo(() => db.vlm_frames.find({
    sort: [{ timestamp: 'desc' }],
    limit: 1
  }), [db]);

  const frames = useRxQuery<any[]>(framesQuery as any);
  const latestFrame = frames && frames.length > 0 ? frames[0].base64Data : null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    let finalUrl = inputUrl;
    if (!finalUrl.startsWith('http')) finalUrl = 'https://' + finalUrl;
    setUrl(finalUrl);
    
    // Also push a navigation intent so the backend VLM browser follows the human
    const newId = crypto.randomUUID();
    db.agent_intents.insert({
      id: newId,
      intentType: 'MANUAL_NAVIGATE',
      payload: { url: finalUrl },
      timestamp: Date.now(),
      status: 'pending'
    }).catch(console.error);
  };

  const handleImageClick = (e: React.MouseEvent<HTMLImageElement>) => {
    // Capture user clicks and send to VLM MCP server via intent
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    
    // Calculate actual scale ratio if the image is scaled down
    const scaleX = e.currentTarget.naturalWidth / rect.width;
    const scaleY = e.currentTarget.naturalHeight / rect.height;
    
    const realX = x * scaleX;
    const realY = y * scaleY;

    const newId = crypto.randomUUID();
    db.agent_intents.insert({
      id: newId,
      intentType: 'MANUAL_CLICK',
      payload: { x: realX, y: realY },
      timestamp: Date.now(),
      status: 'pending'
    }).catch(console.error);
  };

  return (
    <div className="h-full flex flex-col min-w-0 bg-black/20 overflow-hidden">
      <div className="h-14 border-b border-white/5 bg-white/[0.02] flex items-center px-4 gap-4 shadow-sm shrink-0">
        <div className="flex gap-1.5">
          <button className="p-1.5 text-neutral-400 hover:text-white hover:bg-white/10 rounded-md transition-colors"><ChevronLeft className="w-4 h-4" /></button>
          <button className="p-1.5 text-neutral-400 hover:text-white hover:bg-white/10 rounded-md transition-colors"><ChevronRight className="w-4 h-4" /></button>
          <button className="p-1.5 text-neutral-400 hover:text-white hover:bg-white/10 rounded-md transition-colors"><RotateCw className="w-4 h-4" /></button>
        </div>
        <form onSubmit={handleSubmit} className="flex-1 max-w-3xl">
          <input 
            value={inputUrl}
            onChange={e => setInputUrl(e.target.value)}
            className="w-full bg-black/40 border border-white/10 hover:border-white/20 focus:border-accent-blue/50 rounded-full px-5 py-2 text-sm text-neutral-200 outline-none transition-all shadow-inner" 
          />
        </form>
      </div>
      <div className="flex-1 relative bg-neutral-900 flex items-center justify-center overflow-hidden border border-white/5">
         {latestFrame ? (
           <img 
             src={`data:image/jpeg;base64,${latestFrame}`} 
             className="max-w-full max-h-full object-contain cursor-crosshair" 
             alt="VLM Browser Feed"
             onClick={handleImageClick}
           />
         ) : (
           <div className="text-neutral-500 flex flex-col items-center gap-4">
             <BrainCircuit className="w-12 h-12 opacity-50 animate-pulse text-accent-blue" />
             <p>Waiting for VLM Browser Feed...</p>
           </div>
         )}
      </div>
    </div>
  );
}

import { useLayoutStore } from '../hooks/useLayoutStore';

export default function EducationWorkstation() {
  const activeProject = useLayoutStore((state) => state.activeProject);
  if (!activeProject) return <div className="p-4 text-neutral-400 flex items-center justify-center h-full">Please open a project to view the Education Workstation.</div>;

  return (
    <Suspense fallback={<div className="p-4 text-neutral-400">Loading Education Workspace...</div>}>
      <EducationWorkstationContent />
    </Suspense>
  );
}
