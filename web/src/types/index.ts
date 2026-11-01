export type Role = 'director' | 'teacher';
export type UserStatus = 'pending' | 'active' | 'rejected';
export type AttendanceStatus = 'present' | 'absent' | 'late' | 'excused' | 'unmarked';

export interface User {
  id: number;
  school_id: { Int64: number; Valid: boolean };
  telegram_id: number;
  username: { String: string; Valid: boolean };
  full_name: string;
  role: Role;
  status: UserStatus;
  created_at: string;
  updated_at: string;
}

export interface School {
  id: number;
  name: string;
}

export interface ClassItem {
  id: number;
  name: string;
  school_id: number;
  teacher_id: { Int64: number; Valid: boolean };
  teacher_name?: string;
  student_count?: number;
}

export interface TeacherItem {
  id: number;
  full_name: string;
  telegram_id: number;
}

export interface StudentAttendanceItem {
  student_id: number;
  full_name: string;
  class_id: number;
  attendance_id: number;
  status: AttendanceStatus;
  note: string;
}

export interface AttendanceSummary {
  total: number;
  present: number;
  absent: number;
  late: number;
  excused: number;
  unmarked: number;
}

export interface ClassAttendanceData {
  class_id: number;
  date: string;
  is_saved: boolean;
  students: StudentAttendanceItem[];
  summary: AttendanceSummary;
}

export interface UserProfileResponse {
  user: User;
  school?: School;
  settings?: {
    absence_threshold: number;
    deadline_time: string;
  };
  classes?: ClassItem[];
}

export interface GroupedStudent {
  student_id: number;
  student_name: string;
  status: string;
  note: string;
}

export interface ClassGroupedStudents {
  class_id: number;
  class_name: string;
  students: GroupedStudent[];
}

export interface ClassAttendanceStatusItem {
  class_id: number;
  class_name: string;
  teacher_id?: number;
  teacher_name: string;
  total_students: number;
  marked_students: number;
  is_taken: boolean;
}

export interface TodayDashboardResponse {
  date: string;
  total_students: number;
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  attendance_percentage: number;
  absent_list: ClassGroupedStudents[];
  late_list: ClassGroupedStudents[];
  classes_status: ClassAttendanceStatusItem[];
  unmarked_classes_count: number;
}

export interface SettingsData {
  school_id: number;
  absence_threshold: number;
  deadline_time: string;
}

export interface StudentHistoryDayItem {
  date: string;
  day: number;
  status: AttendanceStatus;
  note: string;
}

export interface StudentHistoryResponse {
  student_id: number;
  student_name: string;
  class_name: string;
  year_month: string;
  threshold: number;
  total_days_marked: number;
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  exceeded_threshold: boolean;
  days: StudentHistoryDayItem[];
}

export interface BulkCreateClassesRequest {
  start_grade: number;
  end_grade: number;
  parallels: string[];
}

export interface BulkCreateClassesResponse {
  created_count: number;
  skipped_count: number;
  classes: ClassItem[];
}

export interface TeacherOverviewItem {
  class_id: number;
  class_name: string;
  school_id: number;
  student_count: number;
  primary_record_id?: number | null;
  primary_user_id?: number | null;
  primary_teacher_name?: string | null;
  primary_telegram_id?: number | null;
  primary_user_status?: string | null;
  sub_record_id?: number | null;
  sub_user_id?: number | null;
  sub_teacher_name?: string | null;
  sub_valid_from?: string | null;
  sub_valid_until?: string | null;
  active_invite_token?: string | null;
  active_invite_expires_at?: string | null;
}

export interface AssignTeacherResponse {
  class_id: number;
  teacher?: User;
  invite_token?: string;
  invite_url?: string;
  share_url?: string;
  expires_at?: string;
  message: string;
}

export interface ImportPreviewError {
  row: number;
  sheet: string;
  reason: string;
}

export interface ImportPreviewResponse {
  job_id: string;
  total_rows: number;
  total_classes: number;
  new_students: number;
  existing_students: number;
  duplicate_rows: number;
  error_rows: number;
  missing_classes: string[];
  errors: ImportPreviewError[];
}

export interface ImportJobStatusResponse {
  id: string;
  status: 'pending' | 'running' | 'done' | 'failed';
  total_rows: number;
  processed_rows: number;
  created_count: number;
  updated_count: number;
  skipped_count: number;
  failed_count: number;
  missing_from_file_count: number;
  summary: any;
  error_message?: string;
}

export const getTashkentDate = (d: Date = new Date()): string => {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Tashkent' }).format(d);
};

export const getTashkentMonth = (d: Date = new Date()): string => {
  return getTashkentDate(d).slice(0, 7);
};
