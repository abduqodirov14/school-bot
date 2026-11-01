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
