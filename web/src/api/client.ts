import type {
  AssignTeacherResponse,
  BulkCreateClassesRequest,
  BulkCreateClassesResponse,
  ClassAttendanceData,
  ClassItem,
  ImportJobStatusResponse,
  ImportPreviewResponse,
  SettingsData,
  StudentHistoryResponse,
  TeacherItem,
  TeacherOverviewItem,
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
  getClasses: async () => {
    const res = await apiFetch<ClassItem[]>('/api/classes');
    return Array.isArray(res) ? res : [];
  },
  createClass: (data: { name: string; teacher_id?: number | null }) =>
    apiFetch<ClassItem>('/api/classes', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  bulkCreateClasses: (data: BulkCreateClassesRequest) =>
    apiFetch<BulkCreateClassesResponse>('/api/classes/bulk', {
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
  getTeachers: async () => {
    const res = await apiFetch<TeacherItem[]>('/api/teachers');
    return Array.isArray(res) ? res : [];
  },

  // Class Teachers & Substitutes
  getTeachersOverview: async () => {
    const res = await apiFetch<TeacherOverviewItem[]>('/api/classes/teachers/overview');
    return Array.isArray(res) ? res : [];
  },
  assignTeacher: (classId: number, data: { telegram_id?: number; full_name?: string; generate_invite?: boolean }) =>
    apiFetch<AssignTeacherResponse>(`/api/classes/${classId}/teacher`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  unassignTeacher: (classId: number) =>
    apiFetch<{ message: string }>(`/api/classes/${classId}/teacher`, {
      method: 'DELETE',
    }),
  assignSubstitute: (classId: number, data: { user_id: number; valid_from: string; valid_until?: string }) =>
    apiFetch<any>(`/api/classes/${classId}/substitute`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Excel Importer
  previewImport: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiFetch<ImportPreviewResponse>('/api/import/preview', {
      method: 'POST',
      body: formData,
    });
  },
  commitImport: (jobId: string, createMissingClasses: boolean = true) =>
    apiFetch<{ message: string; job_id: string }>(`/api/import/${jobId}/commit`, {
      method: 'POST',
      body: JSON.stringify({ create_missing_classes: createMissingClasses }),
    }),
  getImportJob: (jobId: string) =>
    apiFetch<ImportJobStatusResponse>(`/api/import/${jobId}`),
  deactivateMissingStudents: (studentIds: number[]) =>
    apiFetch<{ message: string; deactivated_count: number }>('/api/import/deactivate-missing', {
      method: 'POST',
      body: JSON.stringify({ student_ids: studentIds }),
    }),

  // Students
  getStudents: async (classId?: number) => {
    const q = classId ? `?class_id=${classId}` : '';
    const res = await apiFetch<any[]>(`/api/students${q}`);
    return Array.isArray(res) ? res : [];
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
  getTodayDashboard: async (date?: string) => {
    const q = date ? `?date=${date}` : '';
    const res = await apiFetch<TodayDashboardResponse>(`/api/dashboard/today${q}`);
    return {
      ...res,
      absent_list: res.absent_list || [],
      late_list: res.late_list || [],
      classes_status: res.classes_status || [],
    };
  },
  getSettings: () => apiFetch<SettingsData>('/api/settings'),
  updateSettings: (data: { absence_threshold: number; deadline_time: string }) =>
    apiFetch<SettingsData>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  getStudentHistory: async (studentId: number, month?: string) => {
    const q = month ? `?month=${month}` : '';
    const res = await apiFetch<StudentHistoryResponse>(`/api/students/${studentId}/history${q}`);
    return {
      ...res,
      days: Array.isArray(res.days) ? res.days : [],
    };
  },
};
