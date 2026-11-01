import type {
  ClassAttendanceData,
  ClassItem,
  SettingsData,
  StudentHistoryResponse,
  TeacherItem,
  TodayDashboardResponse,
  UserProfileResponse,
} from '../types';

declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        initDataUnsafe: any;
        ready: () => void;
        expand: () => void;
        close: () => void;
        MainButton: any;
        BackButton: any;
        setHeaderColor: (color: string) => void;
        setBackgroundColor: (color: string) => void;
      };
    };
  }
}

export function getInitData(): string {
  if (typeof window !== 'undefined' && window.Telegram?.WebApp?.initData) {
    return window.Telegram.WebApp.initData;
  }
  const urlParams = new URLSearchParams(window.location.search);
  const param = urlParams.get('initData');
  if (param) return param;
  return '';
}

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const initData = getInitData();
  const headers = new Headers(options.headers || {});
  if (initData) {
    headers.set('X-Telegram-Init-Data', initData);
  }
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(path, {
    ...options,
    headers,
  });

  if (!response.ok) {
    let errorMsg = `Xatolik yuz berdi (${response.status})`;
    try {
      const data = await response.json();
      if (data.error) errorMsg = data.error;
    } catch {
      // ignore
    }
    throw new Error(errorMsg);
  }

  return response.json();
}

export const api = {
  // Auth
  getMe: () => apiFetch<UserProfileResponse>('/api/auth/me'),

  // Classes
  getClasses: () => apiFetch<ClassItem[]>('/api/classes'),
  createClass: (data: { name: string; teacher_id?: number | null }) =>
    apiFetch<ClassItem>('/api/classes', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateClass: (id: number, data: { name: string; teacher_id?: number | null }) =>
    apiFetch<ClassItem>(`/api/classes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteClass: (id: number) =>
    apiFetch<{ message: string }>(`/api/classes/${id}`, {
      method: 'DELETE',
    }),
  getTeachers: () => apiFetch<TeacherItem[]>('/api/teachers'),

  // Students
  getStudents: (classId?: number) => {
    const q = classId ? `?class_id=${classId}` : '';
    return apiFetch<any[]>(`/api/students${q}`);
  },
  getStudent: (id: number) => apiFetch<any>(`/api/students/${id}`),
  createStudent: (data: { class_id: number; full_name: string }) =>
    apiFetch<any>('/api/students', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateStudent: (id: number, data: { class_id: number; full_name: string; active: boolean }) =>
    apiFetch<any>(`/api/students/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteStudent: (id: number) =>
    apiFetch<{ message: string }>(`/api/students/${id}`, {
      method: 'DELETE',
    }),
  importStudents: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiFetch<{ imported_students: number; created_classes: number; errors: string[] }>(
      '/api/students/import',
      {
        method: 'POST',
        body: formData,
      }
    );
  },

  // Attendance
  getClassAttendance: (classId: number, date?: string) => {
    const q = new URLSearchParams({ class_id: classId.toString() });
    if (date) q.set('date', date);
    return apiFetch<ClassAttendanceData>(`/api/attendance?${q.toString()}`);
  },
  saveAttendance: (classId: number, date: string, items: { student_id: number; status: string; note: string }[]) => {
    return apiFetch<{ message: string; saved_count: number }>('/api/attendance', {
      method: 'POST',
      body: JSON.stringify({ class_id: classId, date, items }),
    });
  },
  getAttendanceLogs: (studentId: number) => {
    return apiFetch<any[]>(`/api/attendance/logs?student_id=${studentId}`);
  },

  // Director Dashboard & Settings
  getTodayDashboard: (date?: string) => {
    const q = date ? `?date=${date}` : '';
    return apiFetch<TodayDashboardResponse>(`/api/dashboard/today${q}`);
  },
  getSettings: () => apiFetch<SettingsData>('/api/settings'),
  updateSettings: (data: { absence_threshold: number; deadline_time: string }) =>
    apiFetch<SettingsData>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  getStudentHistory: (studentId: number, month?: string) => {
    const q = month ? `?month=${month}` : '';
    return apiFetch<StudentHistoryResponse>(`/api/students/${studentId}/history${q}`);
  },
};
