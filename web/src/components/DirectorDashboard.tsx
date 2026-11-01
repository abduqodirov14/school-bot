import React, { useState, useEffect } from 'react';
import { api, getInitData } from '../api/client';
import { getTashkentDate, getTashkentMonth } from '../types';
import type {
  ClassItem,
  SettingsData,
  StudentHistoryResponse,
  TeacherItem,
  TodayDashboardResponse,
} from '../types';
import { BulkClassModal } from './BulkClassModal';
import { ExcelImportWizard } from './ExcelImportWizard';
import { TeachersOverviewTab } from './TeachersOverviewTab';
import {
  BarChart3,
  Users,
  School,
  Settings,
  AlertTriangle,
  CheckCircle2,
  Calendar,
  Plus,
  Trash2,
  Edit2,
  FileSpreadsheet,
  X,
  Search,
  Clock,
  Save,
  ChevronLeft,
  ChevronRight,
  MessageSquare,
  UserCheck,
  Layers,
} from 'lucide-react';

interface Props {
  schoolName: string;
}

export const DirectorDashboard: React.FC<Props> = ({ schoolName }) => {
  const [activeTab, setActiveTab] = useState<'today' | 'classes' | 'teachers' | 'students' | 'settings'>('today');

  // Today Dashboard State
  const [todayDate, setTodayDate] = useState<string>(getTashkentDate());
  const [dashboard, setDashboard] = useState<TodayDashboardResponse | null>(null);
  const [loadingDashboard, setLoadingDashboard] = useState<boolean>(false);

  // Classes State
  const [classes, setClasses] = useState<ClassItem[]>([]);
  const [teachers, setTeachers] = useState<TeacherItem[]>([]);
  const [loadingClasses, setLoadingClasses] = useState<boolean>(false);
  const [classModalOpen, setClassModalOpen] = useState<boolean>(false);
  const [bulkClassModalOpen, setBulkClassModalOpen] = useState<boolean>(false);
  const [editingClass, setEditingClass] = useState<ClassItem | null>(null);
  const [classNameInput, setClassNameInput] = useState<string>('');
  const [selectedTeacherId, setSelectedTeacherId] = useState<number | null>(null);

  // Students State
  const [students, setStudents] = useState<any[]>([]);
  const [loadingStudents, setLoadingStudents] = useState<boolean>(false);
  const [studentSearch, setStudentSearch] = useState<string>('');
  const [selectedClassFilter, setSelectedClassFilter] = useState<number | null>(null);
  const [studentModalOpen, setStudentModalOpen] = useState<boolean>(false);
  const [studentNameInput, setStudentNameInput] = useState<string>('');
  const [studentClassIdInput, setStudentClassIdInput] = useState<number | null>(null);

  // Import State
  const [importModalOpen, setImportModalOpen] = useState<boolean>(false);

  // Student History Calendar State
  const [historyModalOpen, setHistoryModalOpen] = useState<boolean>(false);
  const [selectedStudentHistory, setSelectedStudentHistory] = useState<StudentHistoryResponse | null>(null);
  const [historyMonth, setHistoryMonth] = useState<string>(getTashkentMonth()); // YYYY-MM
  const [loadingHistory, setLoadingHistory] = useState<boolean>(false);

  // Settings State
  const [settings, setSettings] = useState<SettingsData>({ school_id: 0, absence_threshold: 3, deadline_time: '09:00' });
  const [savingSettings, setSavingSettings] = useState<boolean>(false);
  const [settingsMessage, setSettingsMessage] = useState<string | null>(null);

  // Global Toast
  const [toast, setToast] = useState<string | null>(null);
  const [downloadingReport, setDownloadingReport] = useState<boolean>(false);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 3500);
  };

  const handleDownloadMonthlyExcel = async () => {
    setDownloadingReport(true);
    try {
      const monthStr = todayDate.slice(0, 7);
      const initData = getInitData();
      const headers = new Headers();
      if (initData) headers.set('X-Telegram-Init-Data', initData);

      const resp = await fetch(`/api/reports/monthly?month=${monthStr}`, { headers });
      if (!resp.ok) {
        throw new Error("Hisobotni yuklab bo'lmadi");
      }
      const blob = await resp.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `Davomat_Hisobot_${monthStr}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
      showToast("Excel hisoboti yuklab olindi!");
    } catch (err: any) {
      showToast(err.message || "Yuklab olishda xatolik");
    } finally {
      setDownloadingReport(false);
    }
  };

  useEffect(() => {
    if (activeTab === 'today') loadTodayDashboard(todayDate);
    if (activeTab === 'classes' || activeTab === 'teachers') loadClassesAndTeachers();
    if (activeTab === 'students') {
      loadClassesAndTeachers();
      loadStudents();
    }
    if (activeTab === 'settings') loadSettings();
  }, [activeTab]);

  // Loaders
  const loadTodayDashboard = async (d: string) => {
    setLoadingDashboard(true);
    try {
      const data = await api.getTodayDashboard(d);
      setDashboard(data);
    } catch (err: any) {
      showToast(err.message || 'Hisobotni yuklab bo‘lmadi');
    } finally {
      setLoadingDashboard(false);
    }
  };

  const loadClassesAndTeachers = async () => {
    setLoadingClasses(true);
    try {
      const [cls, tch] = await Promise.all([api.getClasses(), api.getTeachers()]);
      setClasses(Array.isArray(cls) ? cls : []);
      setTeachers(Array.isArray(tch) ? tch : []);
    } catch (err: any) {
      showToast(err.message || 'Sinflarni yuklab bo‘lmadi');
    } finally {
      setLoadingClasses(false);
    }
  };

  const loadStudents = async (classId?: number) => {
    setLoadingStudents(true);
    try {
      const data = await api.getStudents(classId);
      setStudents(Array.isArray(data) ? data : []);
    } catch (err: any) {
      showToast(err.message || 'O‘quvchilarni yuklab bo‘lmadi');
    } finally {
      setLoadingStudents(false);
    }
  };

  const loadSettings = async () => {
    try {
      const data = await api.getSettings();
      setSettings(data);
    } catch (err: any) {
      showToast(err.message || 'Sozlamalarni yuklab bo‘lmadi');
    }
  };

  // Class Actions
  const handleSaveClass = async () => {
    if (!classNameInput.trim()) {
      showToast('Sinf nomi kiritilishi shart');
      return;
    }
    try {
      if (editingClass) {
        await api.updateClass(editingClass.id, {
          name: classNameInput.trim(),
          teacher_id: selectedTeacherId,
        });
        showToast('Sinf yangilandi');
      } else {
        await api.createClass({
          name: classNameInput.trim(),
          teacher_id: selectedTeacherId,
        });
        showToast('Yangi sinf yaratildi');
      }
      setClassModalOpen(false);
      loadClassesAndTeachers();
    } catch (err: any) {
      showToast(err.message || 'Xatolik yuz berdi');
    }
  };

  const handleDeleteClass = async (classId: number) => {
    if (!window.confirm("Rostdan ham ushbu sinfni o'chirmoqchimisiz?")) return;
    try {
      await api.deleteClass(classId);
      showToast('Sinf o‘chirildi');
      loadClassesAndTeachers();
    } catch (err: any) {
      showToast(err.message || 'O‘chirishda xatolik');
    }
  };

  // Student Actions
  const handleSaveStudent = async () => {
    if (!studentNameInput.trim() || !studentClassIdInput) {
      showToast('Ism va sinf tanlanishi shart');
      return;
    }
    try {
      await api.createStudent({
        full_name: studentNameInput.trim(),
        class_id: studentClassIdInput,
      });
      showToast('O‘quvchi qo‘shildi');
      setStudentModalOpen(false);
      loadStudents(selectedClassFilter || undefined);
    } catch (err: any) {
      showToast(err.message || 'Saqlashda xatolik');
    }
  };

  const handleDeleteStudent = async (studentId: number) => {
    if (!window.confirm("O'quvchini safdan chiqarmoqchimisiz?")) return;
    try {
      await api.deleteStudent(studentId);
      showToast('O‘quvchi o‘chirildi');
      loadStudents(selectedClassFilter || undefined);
    } catch (err: any) {
      showToast(err.message || 'Xatolik');
    }
  };

  // Student History Calendar
  const openStudentHistory = async (studentId: number) => {
    setLoadingHistory(true);
    setHistoryModalOpen(true);
    try {
      const data = await api.getStudentHistory(studentId, historyMonth);
      setSelectedStudentHistory(data);
    } catch (err: any) {
      showToast(err.message || 'Tarixni yuklab bo‘lmadi');
    } finally {
      setLoadingHistory(false);
    }
  };

  const changeHistoryMonth = async (direction: number) => {
    if (!selectedStudentHistory) return;
    const [y, m] = historyMonth.split('-').map(Number);
    let newDate = new Date(y, m - 1 + direction, 1);
    const newMonthStr = getTashkentMonth(newDate);
    setHistoryMonth(newMonthStr);
    setLoadingHistory(true);
    try {
      const data = await api.getStudentHistory(selectedStudentHistory.student_id, newMonthStr);
      setSelectedStudentHistory(data);
    } catch (err: any) {
      showToast(err.message || 'Tarixni yuklab bo‘lmadi');
    } finally {
      setLoadingHistory(false);
    }
  };

  // Save Settings
  const handleSaveSettings = async () => {
    setSavingSettings(true);
    setSettingsMessage(null);
    try {
      await api.updateSettings({
        absence_threshold: Number(settings.absence_threshold),
        deadline_time: settings.deadline_time,
      });
      setSettingsMessage('Sozlamalar saqlandi!');
      setTimeout(() => setSettingsMessage(null), 3000);
    } catch (err: any) {
      setSettingsMessage('Saqlashda xatolik: ' + err.message);
    } finally {
      setSavingSettings(false);
    }
  };

  const filteredStudents = (students || []).filter(s => {
    const matchName = s.full_name?.toLowerCase().includes(studentSearch.toLowerCase());
    const matchClass = selectedClassFilter ? s.class_id === selectedClassFilter : true;
    return matchName && matchClass;
  });

  return (
    <div className="min-h-screen bg-slate-50 text-slate-800 pb-20">
      {/* Top Header */}
      <header className="bg-white border-b border-slate-200 px-4 py-3 sticky top-0 z-20 shadow-xs">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-base font-bold text-slate-900 leading-tight">🎓 Maktab Boshqaruvi</h1>
            <p className="text-xs text-slate-500 font-medium">{schoolName}</p>
          </div>
          <span className="text-[11px] font-semibold bg-emerald-100 text-emerald-800 px-2.5 py-1 rounded-full border border-emerald-200">
            Direktor
          </span>
        </div>
      </header>

      {/* Toast Notification */}
      {toast && (
        <div className="fixed top-14 left-1/2 -translate-x-1/2 z-50 bg-slate-900 text-white text-xs font-semibold px-4 py-2 rounded-xl shadow-lg animate-in fade-in slide-in-from-top-2">
          {toast}
        </div>
      )}

      {/* Tab Navigation */}
      <nav className="bg-white border-b border-slate-200 px-1 flex justify-around text-xs font-bold sticky top-14 z-10 overflow-x-auto">
        <button
          onClick={() => setActiveTab('today')}
          className={`py-3 px-2.5 border-b-2 flex items-center gap-1 transition-all shrink-0 ${
            activeTab === 'today'
              ? 'border-emerald-600 text-emerald-600'
              : 'border-transparent text-slate-500 hover:text-slate-800'
          }`}
        >
          <BarChart3 className="w-4 h-4" /> Bugun
        </button>
        <button
          onClick={() => setActiveTab('classes')}
          className={`py-3 px-2.5 border-b-2 flex items-center gap-1 transition-all shrink-0 ${
            activeTab === 'classes'
              ? 'border-emerald-600 text-emerald-600'
              : 'border-transparent text-slate-500 hover:text-slate-800'
          }`}
        >
          <School className="w-4 h-4" /> Sinflar
        </button>
        <button
          onClick={() => setActiveTab('teachers')}
          className={`py-3 px-2.5 border-b-2 flex items-center gap-1 transition-all shrink-0 ${
            activeTab === 'teachers'
              ? 'border-emerald-600 text-emerald-600'
              : 'border-transparent text-slate-500 hover:text-slate-800'
          }`}
        >
          <UserCheck className="w-4 h-4" /> Rahbarlar
        </button>
        <button
          onClick={() => setActiveTab('students')}
          className={`py-3 px-2.5 border-b-2 flex items-center gap-1 transition-all shrink-0 ${
            activeTab === 'students'
              ? 'border-emerald-600 text-emerald-600'
              : 'border-transparent text-slate-500 hover:text-slate-800'
          }`}
        >
          <Users className="w-4 h-4" /> O'quvchilar
        </button>
        <button
          onClick={() => setActiveTab('settings')}
          className={`py-3 px-2.5 border-b-2 flex items-center gap-1 transition-all shrink-0 ${
            activeTab === 'settings'
              ? 'border-emerald-600 text-emerald-600'
              : 'border-transparent text-slate-500 hover:text-slate-800'
          }`}
        >
          <Settings className="w-4 h-4" /> Sozlamalar
        </button>
      </nav>

      {/* Main Content Body */}
      <main className="p-4 max-w-xl mx-auto space-y-4">
        {/* ================= TAB 1: TODAY ================= */}
        {activeTab === 'today' && (
          <div className="space-y-4">
            {/* Date Picker bar */}
            <div className="flex items-center justify-between bg-white p-3 rounded-xl border border-slate-200 shadow-xs">
              <span className="text-xs font-semibold text-slate-600">Davomat sanasi:</span>
              <input
                type="date"
                value={todayDate}
                onChange={e => {
                  setTodayDate(e.target.value);
                  loadTodayDashboard(e.target.value);
                }}
                className="bg-slate-100 border border-slate-300 rounded-lg px-2.5 py-1 text-xs font-semibold text-slate-800"
              />
            </div>

            {/* Excel Report Download Button */}
            <button
              onClick={handleDownloadMonthlyExcel}
              disabled={downloadingReport}
              className="w-full bg-emerald-700 hover:bg-emerald-800 active:scale-98 text-white rounded-xl py-2.5 px-4 text-xs font-bold shadow-xs flex items-center justify-center gap-2 transition-all"
            >
              <FileSpreadsheet className="w-4 h-4" />
              {downloadingReport ? 'Excel hisoboti tayyorlanmoqda...' : `Oylik Excel hisobotni yuklab olish (${todayDate.slice(0, 7)})`}
            </button>

            {loadingDashboard || !dashboard ? (
              <div className="text-center py-12 text-slate-400 text-xs">
                <div className="animate-spin w-6 h-6 border-2 border-emerald-500 border-t-transparent rounded-full mx-auto mb-2" />
                Hisobot yuklanmoqda...
              </div>
            ) : (
              <>
                {/* Unmarked classes warning banner */}
                {dashboard.unmarked_classes_count > 0 && (
                  <div className="bg-amber-50 border border-amber-200 rounded-xl p-3 flex items-start gap-2.5">
                    <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0 mt-0.5" />
                    <div>
                      <p className="text-xs font-bold text-amber-900">
                        {dashboard.unmarked_classes_count} ta sinf hali davomat topshirmadi!
                      </p>
                      <p className="text-[11px] text-amber-700 mt-0.5">
                        {(dashboard.classes_status || [])
                          .filter(c => !c.is_taken && c.total_students > 0)
                          .map(c => c.class_name)
                          .join(', ')}
                      </p>
                    </div>
                  </div>
                )}

                {/* Big Stats Grid */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="bg-white p-3.5 rounded-xl border border-slate-200 shadow-xs">
                    <p className="text-[11px] font-semibold text-slate-500">Jami o'quvchilar</p>
                    <p className="text-2xl font-black text-slate-900 mt-1">{dashboard.total_students}</p>
                    <p className="text-[10px] text-slate-400 mt-1">Maktab bo'yicha</p>
                  </div>

                  <div className="bg-white p-3.5 rounded-xl border border-slate-200 shadow-xs">
                    <p className="text-[11px] font-semibold text-slate-500">Davomat foizi</p>
                    <p className="text-2xl font-black text-emerald-600 mt-1">
                      {dashboard.attendance_percentage}%
                    </p>
                    <p className="text-[10px] text-slate-400 mt-1">{dashboard.present_count} nafar darsda</p>
                  </div>

                  <div className="bg-white p-3.5 rounded-xl border border-rose-200 bg-rose-50/30 shadow-xs">
                    <p className="text-[11px] font-semibold text-rose-700">Kelmadi</p>
                    <p className="text-2xl font-black text-rose-600 mt-1">{dashboard.absent_count}</p>
                    <p className="text-[10px] text-rose-500 mt-1">Sababsiz qoldirgan</p>
                  </div>

                  <div className="bg-white p-3.5 rounded-xl border border-amber-200 bg-amber-50/30 shadow-xs">
                    <p className="text-[11px] font-semibold text-amber-700">Kechikdi / Sababli</p>
                    <p className="text-2xl font-black text-amber-600 mt-1">
                      {dashboard.late_count + dashboard.excused_count}
                    </p>
                    <p className="text-[10px] text-amber-600 mt-1">
                      {dashboard.late_count} kechikdi, {dashboard.excused_count} sababli
                    </p>
                  </div>
                </div>

                {/* Absent students list grouped by class */}
                <div className="bg-white rounded-xl border border-slate-200 shadow-xs overflow-hidden">
                  <div className="p-3 border-b border-slate-100 flex items-center justify-between">
                    <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                      <span className="w-2 h-2 rounded-full bg-rose-500" />
                      Bugun darsga kelmaganlar ({dashboard.absent_count})
                    </h3>
                  </div>
                  {(dashboard.absent_list || []).length === 0 ? (
                    <div className="p-4 text-center text-xs text-slate-400 font-medium">
                      Bugun hech kim dars qoldirmagan! 🎉
                    </div>
                  ) : (
                    <div className="divide-y divide-slate-100">
                      {(dashboard.absent_list || []).map(group => (
                        <div key={group.class_id} className="p-3">
                          <p className="text-xs font-bold text-slate-800 mb-1.5">
                            🏫 {group.class_name} sinfi:
                          </p>
                          <div className="space-y-1 pl-2">
                            {(group.students || []).map((st, i) => (
                              <div key={st.student_id} className="text-xs flex items-center justify-between">
                                <span className="text-slate-700">
                                  {i + 1}. {st.student_name}
                                </span>
                                {st.note && (
                                  <span className="text-[11px] text-slate-400 italic">
                                    Izoh: {st.note}
                                  </span>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Late students list */}
                {(dashboard.late_list || []).length > 0 && (
                  <div className="bg-white rounded-xl border border-slate-200 shadow-xs overflow-hidden">
                    <div className="p-3 border-b border-slate-100">
                      <h3 className="text-xs font-bold text-amber-800 flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5 text-amber-500" />
                        Darsga kechikkanlar ({dashboard.late_count})
                      </h3>
                    </div>
                    <div className="divide-y divide-slate-100">
                      {(dashboard.late_list || []).map(group => (
                        <div key={group.class_id} className="p-3">
                          <p className="text-xs font-bold text-slate-800 mb-1.5">
                            🏫 {group.class_name} sinfi:
                          </p>
                          <div className="space-y-1 pl-2">
                            {(group.students || []).map((st, i) => (
                              <div key={st.student_id} className="text-xs text-slate-700">
                                {i + 1}. {st.student_name}
                              </div>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Class status summary list */}
                <div className="bg-white rounded-xl border border-slate-200 shadow-xs overflow-hidden">
                  <div className="p-3 border-b border-slate-100">
                    <h3 className="text-xs font-bold text-slate-900">Sinflar davomat holati</h3>
                  </div>
                  <div className="divide-y divide-slate-100">
                    {(dashboard.classes_status || []).map(cs => (
                      <div key={cs.class_id} className="p-3 flex items-center justify-between text-xs">
                        <div>
                          <p className="font-bold text-slate-900">{cs.class_name}</p>
                          <p className="text-[11px] text-slate-500">
                            {cs.teacher_name ? `👤 ${cs.teacher_name}` : "O'qituvchi yo'q"}
                          </p>
                        </div>
                        <div className="text-right">
                          {cs.is_taken ? (
                            <span className="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-md border border-emerald-200">
                              <CheckCircle2 className="w-3 h-3" /> Olingan ({cs.marked_students}/{cs.total_students})
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[11px] font-bold text-amber-700 bg-amber-50 px-2 py-0.5 rounded-md border border-amber-200">
                              <AlertTriangle className="w-3 h-3" /> Olinmagan!
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        )}

        {/* ================= TAB 2: CLASSES ================= */}
        {activeTab === 'classes' && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-bold text-slate-700">Mavjud Sinflar ({(classes || []).length})</h2>
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => setBulkClassModalOpen(true)}
                  className="inline-flex items-center gap-1 px-2.5 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-bold border border-slate-200 active:scale-95 transition-all"
                >
                  <Layers className="w-3.5 h-3.5" /> Ommaviy
                </button>
                <button
                  onClick={() => {
                    setEditingClass(null);
                    setClassNameInput('');
                    setSelectedTeacherId(null);
                    setClassModalOpen(true);
                  }}
                  className="inline-flex items-center gap-1 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-bold shadow-xs active:scale-95 transition-all"
                >
                  <Plus className="w-3.5 h-3.5 stroke-[3]" /> Yangi sinf
                </button>
              </div>
            </div>

            {loadingClasses ? (
              <div className="text-center py-12 text-slate-400 text-xs">Yuklanmoqda...</div>
            ) : (classes || []).length === 0 ? (
              <div className="bg-white rounded-xl p-8 text-center text-slate-400 text-xs border border-slate-200">
                Sinflar mavjud emas. Yuqoridagi tugma orqali yarating.
              </div>
            ) : (
              <div className="space-y-2">
                {(classes || []).map(c => (
                  <div
                    key={c.id}
                    className="bg-white rounded-xl p-3 border border-slate-200 shadow-2xs flex items-center justify-between"
                  >
                    <div>
                      <h4 className="text-xs font-bold text-slate-900">{c.name} sinfi</h4>
                      <p className="text-[11px] text-slate-500 mt-0.5">
                        O'qituvchi: <strong>{c.teacher_name || "Biriktirilmagan"}</strong>
                      </p>
                      <p className="text-[10px] text-slate-400 mt-0.5">
                        O'quvchilar: {c.student_count || 0} nafar
                      </p>
                    </div>

                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => {
                          setEditingClass(c);
                          setClassNameInput(c.name);
                          setSelectedTeacherId(c.teacher_id?.Valid ? c.teacher_id.Int64 : null);
                          setClassModalOpen(true);
                        }}
                        className="p-1.5 text-slate-400 hover:text-slate-700 rounded-lg hover:bg-slate-100"
                      >
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteClass(c.id)}
                        className="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg hover:bg-rose-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* ================= TAB 3: TEACHERS (Sinf Rahbarlari) ================= */}
        {activeTab === 'teachers' && (
          <TeachersOverviewTab teachers={teachers} onShowToast={showToast} />
        )}

        {/* ================= TAB 3: STUDENTS ================= */}
        {activeTab === 'students' && (
          <div className="space-y-3">
            {/* Action Bar */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => setImportModalOpen(true)}
                className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 bg-emerald-700 hover:bg-emerald-800 text-white rounded-xl text-xs font-bold shadow-xs active:scale-95 transition-all"
              >
                <FileSpreadsheet className="w-4 h-4" /> Excel Import
              </button>
              <button
                onClick={() => {
                  setStudentNameInput('');
                  setStudentClassIdInput(classes[0]?.id || null);
                  setStudentModalOpen(true);
                }}
                className="inline-flex items-center justify-center gap-1 px-3 py-2 bg-slate-800 hover:bg-slate-900 text-white rounded-xl text-xs font-bold shadow-xs active:scale-95 transition-all"
              >
                <Plus className="w-4 h-4 stroke-[3]" /> Qo'shish
              </button>
            </div>

            {/* Filter and Search */}
            <div className="grid grid-cols-2 gap-2">
              <div className="relative">
                <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-slate-400" />
                <input
                  type="text"
                  placeholder="Ism bo'yicha qidirish..."
                  value={studentSearch}
                  onChange={e => setStudentSearch(e.target.value)}
                  className="w-full pl-8 pr-3 py-1.5 bg-white border border-slate-300 rounded-xl text-xs text-slate-800 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <select
                  value={selectedClassFilter || ''}
                  onChange={e => {
                    const cid = e.target.value ? Number(e.target.value) : null;
                    setSelectedClassFilter(cid);
                    loadStudents(cid || undefined);
                  }}
                  className="w-full py-1.5 px-2 bg-white border border-slate-300 rounded-xl text-xs font-medium text-slate-800 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <option value="">Barcha sinflar</option>
                  {(classes || []).map(c => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Students List */}
            {loadingStudents ? (
              <div className="text-center py-12 text-slate-400 text-xs">Yuklanmoqda...</div>
            ) : (filteredStudents || []).length === 0 ? (
              <div className="bg-white rounded-xl p-8 text-center text-slate-400 text-xs border border-slate-200">
                O'quvchilar topilmadi.
              </div>
            ) : (
              <div className="space-y-2">
                {(filteredStudents || []).map((st, i) => (
                  <div
                    key={st.id}
                    className="bg-white rounded-xl p-3 border border-slate-200 shadow-2xs flex items-center justify-between"
                  >
                    <div>
                      <p className="text-xs font-bold text-slate-900">
                        {i + 1}. {st.full_name}
                      </p>
                      <p className="text-[11px] text-slate-500 mt-0.5">Sinf: {st.class_name}</p>
                    </div>

                    <div className="flex items-center gap-1.5">
                      <button
                        onClick={() => openStudentHistory(st.id)}
                        className="inline-flex items-center gap-1 px-2.5 py-1 bg-slate-100 hover:bg-slate-200 active:scale-95 text-slate-700 rounded-lg text-xs font-semibold"
                      >
                        <Calendar className="w-3 h-3 text-emerald-600" /> Tarix
                      </button>
                      <button
                        onClick={() => handleDeleteStudent(st.id)}
                        className="p-1 text-slate-300 hover:text-rose-600 rounded-lg"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* ================= TAB 4: SETTINGS ================= */}
        {activeTab === 'settings' && (
          <div className="bg-white rounded-xl p-4 border border-slate-200 shadow-xs space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900">Tizim Sozlamalari</h2>
              <p className="text-xs text-slate-500">Davomat muddati va ogohlantirish chegarasi</p>
            </div>

            {settingsMessage && (
              <div className="p-3 bg-emerald-50 text-emerald-800 border border-emerald-200 rounded-xl text-xs font-medium">
                {settingsMessage}
              </div>
            )}

            <div className="space-y-3">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Oylik sababsiz dars qoldirish chegarasi (ogohlantirish uchun):
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    min="1"
                    max="30"
                    value={settings.absence_threshold}
                    onChange={e => setSettings({ ...settings, absence_threshold: Number(e.target.value) })}
                    className="w-24 bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-bold text-slate-900"
                  />
                  <span className="text-xs text-slate-500">kun / oyiga (standart: 3 ta)</span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Davomat topshirish muddati (Deadline):
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="time"
                    value={settings.deadline_time}
                    onChange={e => setSettings({ ...settings, deadline_time: e.target.value })}
                    className="w-32 bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-bold text-slate-900"
                  />
                  <span className="text-xs text-slate-500">(standart: 09:00)</span>
                </div>
              </div>

              <button
                onClick={handleSaveSettings}
                disabled={savingSettings}
                className="w-full bg-emerald-600 hover:bg-emerald-700 active:scale-98 text-white font-bold py-2.5 px-4 rounded-xl text-xs flex items-center justify-center gap-2 transition-all mt-4"
              >
                <Save className="w-4 h-4 stroke-[2.5]" />
                {savingSettings ? 'Saqlanmoqda...' : 'Sozlamalarni saqlash'}
              </button>
            </div>
          </div>
        )}
      </main>

      {/* ================= MODAL: CREATE/EDIT CLASS ================= */}
      {classModalOpen && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-3 z-50">
          <div className="bg-white w-full max-w-sm rounded-2xl p-4 shadow-xl border border-slate-200 animate-in fade-in">
            <div className="flex justify-between items-center mb-3">
              <h3 className="text-sm font-bold text-slate-900">
                {editingClass ? 'Sinfni tahrirlash' : 'Yangi sinf yaratish'}
              </h3>
              <button onClick={() => setClassModalOpen(false)} className="text-slate-400 p-1">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-3 mb-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Sinf nomi:</label>
                <input
                  type="text"
                  placeholder="Masalan: 7-A"
                  value={classNameInput}
                  onChange={e => setClassNameInput(e.target.value)}
                  className="w-full bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-semibold text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Biriktirilgan o'qituvchi:</label>
                <select
                  value={selectedTeacherId || ''}
                  onChange={e => setSelectedTeacherId(e.target.value ? Number(e.target.value) : null)}
                  className="w-full bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-medium text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <option value="">O'qituvchi biriktirilmasin</option>
                  {(teachers || []).map(t => (
                    <option key={t.id} value={t.id}>
                      {t.full_name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <button
              onClick={handleSaveClass}
              className="w-full bg-emerald-600 hover:bg-emerald-700 text-white font-bold py-2.5 rounded-xl text-xs"
            >
              Saqlash
            </button>
          </div>
        </div>
      )}

      {/* ================= MODAL: ADD STUDENT ================= */}
      {studentModalOpen && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-3 z-50">
          <div className="bg-white w-full max-w-sm rounded-2xl p-4 shadow-xl border border-slate-200 animate-in fade-in">
            <div className="flex justify-between items-center mb-3">
              <h3 className="text-sm font-bold text-slate-900">Yangi o'quvchi qo'shish</h3>
              <button onClick={() => setStudentModalOpen(false)} className="text-slate-400 p-1">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-3 mb-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">O'quvchi F.I.SH:</label>
                <input
                  type="text"
                  placeholder="Familiya Ism Sharif"
                  value={studentNameInput}
                  onChange={e => setStudentNameInput(e.target.value)}
                  className="w-full bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-semibold text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Sinf:</label>
                <select
                  value={studentClassIdInput || ''}
                  onChange={e => setStudentClassIdInput(Number(e.target.value))}
                  className="w-full bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs font-medium text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  {(classes || []).map(c => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <button
              onClick={handleSaveStudent}
              className="w-full bg-emerald-600 hover:bg-emerald-700 text-white font-bold py-2.5 rounded-xl text-xs"
            >
              Qo'shish
            </button>
          </div>
        </div>
      )}

      {/* ================= MODAL: EXCEL IMPORT WIZARD ================= */}
      <ExcelImportWizard
        isOpen={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onSuccess={msg => {
          showToast(msg);
          loadClassesAndTeachers();
          loadStudents();
        }}
      />

      {/* ================= MODAL: BULK CLASS CREATION ================= */}
      <BulkClassModal
        isOpen={bulkClassModalOpen}
        onClose={() => setBulkClassModalOpen(false)}
        onSuccess={msg => {
          showToast(msg);
          loadClassesAndTeachers();
        }}
      />

      {/* ================= MODAL: STUDENT ATTENDANCE CALENDAR ================= */}
      {historyModalOpen && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-xs flex items-center justify-center p-3 z-50">
          <div className="bg-white w-full max-w-sm rounded-2xl p-4 shadow-xl border border-slate-200 animate-in fade-in max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-3">
              <div>
                <h3 className="text-sm font-bold text-slate-900">
                  {selectedStudentHistory?.student_name}
                </h3>
                <p className="text-xs text-slate-500">
                  Sinf: {selectedStudentHistory?.class_name} | Davomat Kalendari
                </p>
              </div>
              <button onClick={() => setHistoryModalOpen(false)} className="text-slate-400 p-1">
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Threshold Exceeded Warning */}
            {selectedStudentHistory?.exceeded_threshold && (
              <div className="bg-rose-50 border border-rose-200 rounded-xl p-2.5 mb-3 flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
                <p className="text-[11px] font-bold text-rose-800 leading-tight">
                  Ogohlantirish: Oylik sababsiz qoldirish chegarasiga ({selectedStudentHistory.threshold} ta) yetgan yoki oshgan! (Jami: {selectedStudentHistory.absent_count} kun)
                </p>
              </div>
            )}

            {/* Month Selector */}
            <div className="flex items-center justify-between bg-slate-100 rounded-xl px-3 py-1.5 mb-3">
              <button onClick={() => changeHistoryMonth(-1)} className="p-1 hover:bg-slate-200 rounded-md">
                <ChevronLeft className="w-4 h-4" />
              </button>
              <span className="text-xs font-bold text-slate-700">{historyMonth}</span>
              <button onClick={() => changeHistoryMonth(1)} className="p-1 hover:bg-slate-200 rounded-md">
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>

            {loadingHistory ? (
              <div className="text-center py-8 text-xs text-slate-400">Yuklanmoqda...</div>
            ) : !selectedStudentHistory ? (
              <div className="text-center py-8 text-xs text-slate-400">Ma'lumot topilmadi</div>
            ) : (
              <>
                {/* Monthly Summary Badges */}
                <div className="grid grid-cols-4 gap-1.5 text-center text-xs mb-3">
                  <div className="bg-emerald-50 border border-emerald-100 rounded-lg py-1">
                    <span className="block font-bold text-emerald-700">{selectedStudentHistory.present_count}</span>
                    <span className="text-[10px] text-emerald-600">Keldi</span>
                  </div>
                  <div className="bg-rose-50 border border-rose-100 rounded-lg py-1">
                    <span className="block font-bold text-rose-700">{selectedStudentHistory.absent_count}</span>
                    <span className="text-[10px] text-rose-600">Kelmadi</span>
                  </div>
                  <div className="bg-amber-50 border border-amber-100 rounded-lg py-1">
                    <span className="block font-bold text-amber-700">{selectedStudentHistory.late_count}</span>
                    <span className="text-[10px] text-amber-600">Kechikdi</span>
                  </div>
                  <div className="bg-sky-50 border border-sky-100 rounded-lg py-1">
                    <span className="block font-bold text-sky-700">{selectedStudentHistory.excused_count}</span>
                    <span className="text-[10px] text-sky-600">Sababli</span>
                  </div>
                </div>

                {/* Calendar Days Grid */}
                <div className="space-y-1.5 max-h-56 overflow-y-auto pr-1">
                  {(selectedStudentHistory.days || []).length === 0 ? (
                    <p className="text-center py-4 text-xs text-slate-400">Ushbu oyda davomat qayd etilmagan</p>
                  ) : (
                    (selectedStudentHistory.days || []).map(d => {
                      let colorClass = 'bg-slate-100 text-slate-600';
                      let statusText = 'Belgilanmagan';
                      if (d.status === 'present') {
                        colorClass = 'bg-emerald-100 text-emerald-800 border-emerald-200';
                        statusText = 'Keldi (+)';
                      } else if (d.status === 'absent') {
                        colorClass = 'bg-rose-100 text-rose-800 border-rose-200 font-bold';
                        statusText = 'Kelmadi (-)';
                      } else if (d.status === 'late') {
                        colorClass = 'bg-amber-100 text-amber-800 border-amber-200';
                        statusText = 'Kechikdi (K)';
                      } else if (d.status === 'excused') {
                        colorClass = 'bg-sky-100 text-sky-800 border-sky-200';
                        statusText = 'Sababli (S)';
                      }

                      return (
                        <div
                          key={d.date}
                          className="flex items-center justify-between text-xs p-2 rounded-lg bg-slate-50 border border-slate-100"
                        >
                          <span className="font-semibold text-slate-700">{d.date}</span>
                          <div className="flex items-center gap-2">
                            {d.note && (
                              <span className="text-[10px] text-slate-400 flex items-center gap-0.5">
                                <MessageSquare className="w-3 h-3 text-sky-500" /> {d.note}
                              </span>
                            )}
                            <span className={`px-2 py-0.5 rounded-md text-[11px] border ${colorClass}`}>
                              {statusText}
                            </span>
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
