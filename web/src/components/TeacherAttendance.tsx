import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import { getTashkentDate } from '../types';
import type { AttendanceStatus, ClassItem, StudentAttendanceItem } from '../types';
import { Check, X, Clock, FileText, CheckCircle2, AlertCircle, Save, HelpCircle, MessageSquare } from 'lucide-react';

interface Props {
  initialClasses: ClassItem[];
  schoolName: string;
}

export const TeacherAttendance: React.FC<Props> = ({ initialClasses, schoolName }) => {
  const [classes] = useState<ClassItem[]>(initialClasses);
  const [selectedClassId, setSelectedClassId] = useState<number>(initialClasses[0]?.id || 0);
  const [date, setDate] = useState<string>(getTashkentDate());
  const [students, setStudents] = useState<StudentAttendanceItem[]>([]);
  const [isSaved, setIsSaved] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [saving, setSaving] = useState<boolean>(false);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  // Modal for editing status and note
  const [editingStudent, setEditingStudent] = useState<StudentAttendanceItem | null>(null);
  const [modalNote, setModalNote] = useState<string>('');

  useEffect(() => {
    if (selectedClassId) {
      loadAttendance(selectedClassId, date);
    }
  }, [selectedClassId, date]);

  const loadAttendance = async (classId: number, targetDate: string) => {
    setLoading(true);
    setMessage(null);
    try {
      const data = await api.getClassAttendance(classId, targetDate);
      setStudents(data.students || []);
      setIsSaved(data.is_saved);
    } catch (err: any) {
      setMessage({ text: err.message || "Davomatni yuklab bo'lmadi", type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleAllPresent = () => {
    setStudents(prev =>
      prev.map(s => ({
        ...s,
        status: 'present',
      }))
    );
    setMessage({ text: "Barcha o'quvchilar 'Keldi' qilindi. Saqlashni unutmang!", type: 'success' });
  };

  const cycleStatus = (current: AttendanceStatus): AttendanceStatus => {
    switch (current) {
      case 'unmarked':
        return 'present';
      case 'present':
        return 'absent';
      case 'absent':
        return 'late';
      case 'late':
        return 'excused';
      case 'excused':
        return 'present';
      default:
        return 'present';
    }
  };

  const handleStudentClick = (studentId: number) => {
    setStudents(prev =>
      prev.map(s => {
        if (s.student_id === studentId) {
          return {
            ...s,
            status: cycleStatus(s.status),
          };
        }
        return s;
      })
    );
  };

  const openEditModal = (student: StudentAttendanceItem, e: React.MouseEvent) => {
    e.stopPropagation();
    setEditingStudent(student);
    setModalNote(student.note || '');
  };

  const applyModalChanges = (newStatus: AttendanceStatus) => {
    if (!editingStudent) return;
    setStudents(prev =>
      prev.map(s => {
        if (s.student_id === editingStudent.student_id) {
          return {
            ...s,
            status: newStatus,
            note: modalNote.trim(),
          };
        }
        return s;
      })
    );
    setEditingStudent(null);
  };

  const handleSave = async () => {
    if (!selectedClassId) return;
    setSaving(true);
    setMessage(null);
    try {
      const items = students.map(s => ({
        student_id: s.student_id,
        status: s.status,
        note: s.note || '',
      }));
      await api.saveAttendance(selectedClassId, date, items);
      setIsSaved(true);
      setMessage({ text: "Davomat muvaffaqiyatli saqlandi!", type: 'success' });
      // Clear success notification after 4 seconds
      setTimeout(() => setMessage(null), 4000);
    } catch (err: any) {
      setMessage({ text: err.message || "Saqlashda xatolik yuz berdi", type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  // Calculations
  const counts = students.reduce(
    (acc, s) => {
      acc[s.status] = (acc[s.status] || 0) + 1;
      return acc;
    },
    { present: 0, absent: 0, late: 0, excused: 0, unmarked: 0 } as Record<AttendanceStatus, number>
  );

  const getStatusBadge = (status: AttendanceStatus) => {
    switch (status) {
      case 'present':
        return (
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800 border border-emerald-200 shadow-xs">
            <Check className="w-3.5 h-3.5" /> Keldi
          </span>
        );
      case 'absent':
        return (
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-800 border border-rose-200 shadow-xs">
            <X className="w-3.5 h-3.5" /> Kelmadi
          </span>
        );
      case 'late':
        return (
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800 border border-amber-200 shadow-xs">
            <Clock className="w-3.5 h-3.5" /> Kechikdi
          </span>
        );
      case 'excused':
        return (
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold bg-sky-100 text-sky-800 border border-sky-200 shadow-xs">
            <FileText className="w-3.5 h-3.5" /> Sababli
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600 border border-slate-200">
            <HelpCircle className="w-3.5 h-3.5" /> Belgilanmagan
          </span>
        );
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 pb-28 text-slate-800">
      {/* Top Header */}
      <header className="bg-white border-b border-slate-200 px-4 py-3 sticky top-0 z-20 shadow-xs">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h1 className="text-base font-bold text-slate-900 leading-tight">📋 Davomat Jurnali</h1>
            <p className="text-xs text-slate-500">{schoolName}</p>
          </div>
          <div>
            {isSaved ? (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-emerald-700 bg-emerald-50 px-2.5 py-1 rounded-full border border-emerald-200">
                <CheckCircle2 className="w-3.5 h-3.5" /> Saqlangan
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-amber-700 bg-amber-50 px-2.5 py-1 rounded-full border border-amber-200">
                <AlertCircle className="w-3.5 h-3.5" /> Olinmagan
              </span>
            )}
          </div>
        </div>

        {/* Class and Date Selector */}
        <div className="grid grid-cols-2 gap-2 pt-1">
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">Sinf</label>
            <select
              value={selectedClassId}
              onChange={e => setSelectedClassId(Number(e.target.value))}
              className="w-full bg-slate-100 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            >
              {classes.map(c => (
                <option key={c.id} value={c.id}>
                  {c.name} sinfi
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">Sana</label>
            <input
              type="date"
              value={date}
              onChange={e => setDate(e.target.value)}
              className="w-full bg-slate-100 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="p-4 max-w-xl mx-auto space-y-3">
        {/* Top Quick Actions & Stats */}
        <div className="bg-white rounded-xl p-3 border border-slate-200 shadow-xs space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-600">
              Jami: <strong>{students.length}</strong> nafar o'quvchi
            </span>
            <button
              onClick={handleAllPresent}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 active:scale-95 text-white text-xs font-bold rounded-lg shadow-sm transition-all"
            >
              <Check className="w-3.5 h-3.5 stroke-[3]" /> Hammasi keldi
            </button>
          </div>

          {/* Counts pill badges */}
          <div className="grid grid-cols-4 gap-1.5 text-center text-xs">
            <div className="bg-emerald-50 border border-emerald-100 rounded-lg py-1.5">
              <span className="block font-bold text-emerald-700 text-sm">{counts.present}</span>
              <span className="text-[10px] text-emerald-600">Keldi</span>
            </div>
            <div className="bg-rose-50 border border-rose-100 rounded-lg py-1.5">
              <span className="block font-bold text-rose-700 text-sm">{counts.absent}</span>
              <span className="text-[10px] text-rose-600">Kelmadi</span>
            </div>
            <div className="bg-amber-50 border border-amber-100 rounded-lg py-1.5">
              <span className="block font-bold text-amber-700 text-sm">{counts.late}</span>
              <span className="text-[10px] text-amber-600">Kechikdi</span>
            </div>
            <div className="bg-sky-50 border border-sky-100 rounded-lg py-1.5">
              <span className="block font-bold text-sky-700 text-sm">{counts.excused}</span>
              <span className="text-[10px] text-sky-600">Sababli</span>
            </div>
          </div>
        </div>

        {/* Status message */}
        {message && (
          <div
            className={`p-3 rounded-lg text-xs font-medium border flex items-center gap-2 ${
              message.type === 'success'
                ? 'bg-emerald-50 text-emerald-800 border-emerald-200'
                : 'bg-rose-50 text-rose-800 border-rose-200'
            }`}
          >
            {message.type === 'success' ? <CheckCircle2 className="w-4 h-4 shrink-0" /> : <AlertCircle className="w-4 h-4 shrink-0" />}
            <span>{message.text}</span>
          </div>
        )}

        {/* Student Attendance List */}
        {loading ? (
          <div className="text-center py-12 text-slate-400 text-xs">
            <div className="animate-spin w-6 h-6 border-2 border-emerald-500 border-t-transparent rounded-full mx-auto mb-2" />
            Yuklanmoqda...
          </div>
        ) : students.length === 0 ? (
          <div className="bg-white rounded-xl p-8 text-center text-slate-500 border border-slate-200 text-xs">
            Ushbu sinfda o'quvchilar ro'yxati mavjud emas.
          </div>
        ) : (
          <div className="space-y-2">
            <p className="text-[11px] text-slate-500 px-1">
              💡 O'quvchi ustiga bosib holatini o'zgartiring (Keldi ➔ Kelmadi ➔ Kechikdi ➔ Sababli):
            </p>
            {students.map((student, index) => (
              <div
                key={student.student_id}
                onClick={() => handleStudentClick(student.student_id)}
                className="bg-white rounded-xl p-3 border border-slate-200 hover:border-slate-300 active:bg-slate-50 cursor-pointer shadow-2xs flex items-center justify-between gap-2 transition-all"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span className="w-6 text-center text-xs font-bold text-slate-400 shrink-0">
                    {index + 1}
                  </span>
                  <div className="min-w-0">
                    <p className="text-xs font-bold text-slate-900 truncate">
                      {student.full_name}
                    </p>
                    {student.note && (
                      <p className="text-[11px] text-slate-500 flex items-center gap-1 mt-0.5 truncate">
                        <MessageSquare className="w-3 h-3 text-sky-500 shrink-0" />
                        {student.note}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  {getStatusBadge(student.status)}
                  <button
                    onClick={(e) => openEditModal(student, e)}
                    title="Izoh qo'shish yoki holatni tanlash"
                    className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 active:scale-95 transition-all"
                  >
                    <MessageSquare className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Floating Bottom Save Bar */}
      <div className="fixed bottom-0 inset-x-0 bg-white/95 backdrop-blur-md border-t border-slate-200 p-3 z-20">
        <div className="max-w-xl mx-auto flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving || loading || students.length === 0}
            className="flex-1 bg-emerald-600 hover:bg-emerald-700 active:scale-98 disabled:opacity-50 text-white font-bold py-3 px-4 rounded-xl shadow-md flex items-center justify-center gap-2 text-sm transition-all"
          >
            <Save className="w-4 h-4 stroke-[2.5]" />
            {saving ? 'Saqlanmoqda...' : 'Saqlash'}
          </button>
        </div>
      </div>

      {/* Modal for detailed status selection and note */}
      {editingStudent && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-3 z-50">
          <div className="bg-white w-full max-w-sm rounded-2xl p-4 shadow-xl border border-slate-200 animate-in fade-in slide-in-from-bottom duration-200">
            <div className="flex justify-between items-center mb-3">
              <div>
                <h3 className="text-sm font-bold text-slate-900">{editingStudent.full_name}</h3>
                <p className="text-xs text-slate-500">Holatni tanlang va izoh qoldiring</p>
              </div>
              <button
                onClick={() => setEditingStudent(null)}
                className="text-slate-400 hover:text-slate-600 p-1 rounded-lg"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Note input */}
            <div className="mb-4">
              <label className="block text-xs font-medium text-slate-600 mb-1">Izoh (ixtiyoriy):</label>
              <input
                type="text"
                placeholder="Masalan: Shamollagan, shifokor qabuli..."
                value={modalNote}
                onChange={e => setModalNote(e.target.value)}
                className="w-full bg-slate-100 border border-slate-300 rounded-xl px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-emerald-500 text-slate-800"
              />
            </div>

            {/* Status options grid */}
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => applyModalChanges('present')}
                className="flex items-center justify-center gap-2 py-2.5 px-3 rounded-xl bg-emerald-50 text-emerald-800 border border-emerald-200 font-bold text-xs hover:bg-emerald-100 active:scale-95 transition-all"
              >
                <Check className="w-4 h-4 text-emerald-600 stroke-[3]" /> Keldi
              </button>

              <button
                onClick={() => applyModalChanges('absent')}
                className="flex items-center justify-center gap-2 py-2.5 px-3 rounded-xl bg-rose-50 text-rose-800 border border-rose-200 font-bold text-xs hover:bg-rose-100 active:scale-95 transition-all"
              >
                <X className="w-4 h-4 text-rose-600 stroke-[3]" /> Kelmadi
              </button>

              <button
                onClick={() => applyModalChanges('late')}
                className="flex items-center justify-center gap-2 py-2.5 px-3 rounded-xl bg-amber-50 text-amber-800 border border-amber-200 font-bold text-xs hover:bg-amber-100 active:scale-95 transition-all"
              >
                <Clock className="w-4 h-4 text-amber-600 stroke-[2.5]" /> Kechikdi
              </button>

              <button
                onClick={() => applyModalChanges('excused')}
                className="flex items-center justify-center gap-2 py-2.5 px-3 rounded-xl bg-sky-50 text-sky-800 border border-sky-200 font-bold text-xs hover:bg-sky-100 active:scale-95 transition-all"
              >
                <FileText className="w-4 h-4 text-sky-600 stroke-[2.5]" /> Sababli
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
