import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import type { TeacherItem, TeacherOverviewItem } from '../types';
import {
  UserPlus,
  Share2,
  Copy,
  Trash2,
  Calendar,
  X,
  Check,
  Clock,
} from 'lucide-react';

interface Props {
  teachers: TeacherItem[];
  onShowToast: (msg: string) => void;
}

export const TeachersOverviewTab: React.FC<Props> = ({ teachers, onShowToast }) => {
  const [items, setItems] = useState<TeacherOverviewItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  // Invite Modal
  const [inviteModalOpen, setInviteModalOpen] = useState<boolean>(false);
  const [currentClass, setCurrentClass] = useState<TeacherOverviewItem | null>(null);
  const [inviteURL, setInviteURL] = useState<string>('');
  const [shareURL, setShareURL] = useState<string>('');
  const [generatingInvite, setGeneratingInvite] = useState<boolean>(false);

  // Substitute Modal
  const [subModalOpen, setSubModalOpen] = useState<boolean>(false);
  const [subUserId, setSubUserId] = useState<number | null>(null);
  const [subValidFrom, setSubValidFrom] = useState<string>('');
  const [subValidUntil, setSubValidUntil] = useState<string>('');
  const [savingSub, setSavingSub] = useState<boolean>(false);

  // Direct Assign Modal
  const [assignModalOpen, setAssignModalOpen] = useState<boolean>(false);
  const [assignTelegramId, setAssignTelegramId] = useState<string>('');
  const [assignFullName, setAssignFullName] = useState<string>('');
  const [savingAssign, setSavingAssign] = useState<boolean>(false);

  const loadOverview = async () => {
    setLoading(true);
    try {
      const data = await api.getTeachersOverview();
      setItems(Array.isArray(data) ? data : []);
    } catch (err: any) {
      onShowToast(err.message || "Sinf rahbarlarini yuklab bo'lmadi");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOverview();
  }, []);

  const handleOpenInvite = async (item: TeacherOverviewItem) => {
    setCurrentClass(item);
    setGeneratingInvite(true);
    setInviteModalOpen(true);
    try {
      const res = await api.assignTeacher(item.class_id, { generate_invite: true });
      setInviteURL(res.invite_url || '');
      setShareURL(res.share_url || '');
      loadOverview();
    } catch (err: any) {
      onShowToast(err.message || "Taklif havolasini yaratishda xatolik");
    } finally {
      setGeneratingInvite(false);
    }
  };

  const handleCopyLink = () => {
    if (!inviteURL) return;
    navigator.clipboard.writeText(inviteURL);
    onShowToast("Taklif havolasi nusxalandi!");
  };

  const handleUnassignPrimary = async (classId: number) => {
    if (!window.confirm("Rostdan ham ushbu sinf rahbarini vazifasidan ozod qilmoqchimisiz?")) return;
    try {
      await api.unassignTeacher(classId);
      onShowToast("Sinf rahbari vazifasidan ozod qilindi");
      loadOverview();
    } catch (err: any) {
      onShowToast(err.message || "Xatolik yuz berdi");
    }
  };

  const handleOpenSubModal = (item: TeacherOverviewItem) => {
    setCurrentClass(item);
    setSubUserId(teachers[0]?.id || null);
    const today = new Date().toISOString().slice(0, 10);
    setSubValidFrom(today);
    setSubValidUntil(today);
    setSubModalOpen(true);
  };

  const handleSaveSubstitute = async () => {
    if (!currentClass || !subUserId || !subValidFrom) {
      onShowToast("O'qituvchi va boshlanish sanasi tanlanishi kerak");
      return;
    }

    setSavingSub(true);
    try {
      await api.assignSubstitute(currentClass.class_id, {
        user_id: subUserId,
        valid_from: subValidFrom,
        valid_until: subValidUntil || undefined,
      });
      onShowToast("Vaqtinchalik o'rinbosar biriktirildi!");
      setSubModalOpen(false);
      loadOverview();
    } catch (err: any) {
      onShowToast(err.message || "O'rinbosarni biriktirishda xatolik");
    } finally {
      setSavingSub(false);
    }
  };

  const handleOpenAssignModal = (item: TeacherOverviewItem) => {
    setCurrentClass(item);
    setAssignTelegramId('');
    setAssignFullName('');
    setAssignModalOpen(true);
  };

  const handleSaveDirectAssign = async () => {
    if (!currentClass || !assignTelegramId) {
      onShowToast("Telegram ID kiritilishi kerak");
      return;
    }

    const tgId = parseInt(assignTelegramId.trim(), 10);
    if (isNaN(tgId) || tgId <= 0) {
      onShowToast("Yaroqsiz Telegram ID");
      return;
    }

    setSavingAssign(true);
    try {
      await api.assignTeacher(currentClass.class_id, {
        telegram_id: tgId,
        full_name: assignFullName.trim() || undefined,
      });
      onShowToast("Sinf rahbari muvaffaqiyatli biriktirildi!");
      setAssignModalOpen(false);
      loadOverview();
    } catch (err: any) {
      onShowToast(err.message || "Biriktirishda xatolik");
    } finally {
      setSavingAssign(false);
    }
  };

  return (
    <div className="space-y-3 text-slate-800">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xs font-bold text-slate-700">Sinf Rahbarlari ({items.length})</h2>
          <p className="text-[10px] text-slate-400">Har bir sinf uchun asosiy rahbar yoki o'rinbosar tayinlang</p>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-slate-400 text-xs">Yuklanmoqda...</div>
      ) : items.length === 0 ? (
        <div className="bg-white rounded-xl p-8 text-center text-slate-400 text-xs border border-slate-200">
          Sinflar mavjud emas. Avval sinflarni yarating.
        </div>
      ) : (
        <div className="space-y-2.5">
          {items.map(item => {
            const hasPrimary = !!item.primary_user_id;
            const isInvited = item.primary_user_status === 'invited';
            const hasSub = !!item.sub_user_id;

            return (
              <div
                key={item.class_id}
                className="bg-white rounded-xl p-3 border border-slate-200 shadow-2xs space-y-2.5"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-bold text-slate-900">{item.class_name} sinfi</h4>
                      <span className="text-[10px] bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full font-semibold">
                        {item.student_count} nafar o'quvchi
                      </span>
                    </div>

                    {/* Primary Teacher Display */}
                    <div className="mt-1 flex items-center gap-2 text-xs">
                      {hasPrimary ? (
                        <>
                          <span className="font-semibold text-slate-800">
                            {item.primary_teacher_name || `ID: ${item.primary_telegram_id}`}
                          </span>
                          {isInvited ? (
                            <span className="inline-flex items-center gap-1 text-[10px] font-bold text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
                              <Clock className="w-3 h-3" /> Taklif kutilmoqda
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                              <Check className="w-3 h-3" /> Faol rahbar
                            </span>
                          )}
                        </>
                      ) : (
                        <span className="text-slate-400 italic">Sinf rahbari tayinlanmagan</span>
                      )}
                    </div>

                    {/* Substitute Teacher Display */}
                    {hasSub && (
                      <div className="mt-1 text-[11px] text-purple-700 bg-purple-50 p-1.5 rounded-lg border border-purple-200 flex items-center gap-1.5">
                        <Calendar className="w-3.5 h-3.5 shrink-0" />
                        <span>
                          <strong>O'rinbosar:</strong> {item.sub_teacher_name} ({item.sub_valid_from} dan {item.sub_valid_until || "noma'lumgacha"})
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Top Action Icons */}
                  <div className="flex items-center gap-1">
                    {hasPrimary && (
                      <button
                        onClick={() => handleUnassignPrimary(item.class_id)}
                        title="Rahbarni vazifasidan ozod qilish"
                        className="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg hover:bg-rose-50 transition-all"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                </div>

                {/* Bottom Action Buttons */}
                <div className="grid grid-cols-3 gap-1.5 pt-1 border-t border-slate-100">
                  <button
                    onClick={() => handleOpenInvite(item)}
                    className="inline-flex items-center justify-center gap-1 px-2 py-1.5 bg-emerald-50 hover:bg-emerald-100 text-emerald-700 rounded-lg text-xs font-bold border border-emerald-200 active:scale-95 transition-all truncate"
                  >
                    <Share2 className="w-3.5 h-3.5 shrink-0" /> Havola
                  </button>

                  <button
                    onClick={() => handleOpenAssignModal(item)}
                    className="inline-flex items-center justify-center gap-1 px-2 py-1.5 bg-blue-50 hover:bg-blue-100 text-blue-700 rounded-lg text-xs font-bold border border-blue-200 active:scale-95 transition-all truncate"
                  >
                    <UserPlus className="w-3.5 h-3.5 shrink-0" /> ID orqali
                  </button>

                  <button
                    onClick={() => handleOpenSubModal(item)}
                    className="inline-flex items-center justify-center gap-1 px-2 py-1.5 bg-purple-50 hover:bg-purple-100 text-purple-700 rounded-lg text-xs font-bold border border-purple-200 active:scale-95 transition-all truncate"
                  >
                    <Calendar className="w-3.5 h-3.5 shrink-0" /> O'rinbosar
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* INVITE MODAL */}
      {inviteModalOpen && currentClass && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl w-full max-w-sm p-4 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150 space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-700 flex items-center justify-center">
                  <Share2 className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-slate-800">Taklif havolasi</h3>
                  <p className="text-[10px] text-slate-400">{currentClass.class_name} sinfi uchun</p>
                </div>
              </div>
              <button onClick={() => setInviteModalOpen(false)} className="text-slate-400 hover:text-slate-600 p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            {generatingInvite ? (
              <div className="py-8 text-center text-xs text-slate-400">
                Havola yaratilmoqda...
              </div>
            ) : (
              <div className="space-y-3">
                <div className="p-3 bg-slate-50 border border-slate-200 rounded-xl text-xs break-all font-mono text-slate-700 select-all">
                  {inviteURL}
                </div>

                <p className="text-[11px] text-slate-500">
                  Ushbu havola <strong>7 kun</strong> davomida amal qiladi va faqat <strong>1 marta</strong> ishlatilishi mumkin. O'qituvchi ushbu havola orqali botga kirganida, avtomatik ravishda sinf rahbari sifatida biriktiriladi.
                </p>

                <div className="grid grid-cols-2 gap-2">
                  <button
                    onClick={handleCopyLink}
                    className="flex items-center justify-center gap-1.5 py-2 px-3 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-xl text-xs font-bold active:scale-95 transition-all"
                  >
                    <Copy className="w-3.5 h-3.5" /> Nusxalash
                  </button>

                  <a
                    href={shareURL}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center justify-center gap-1.5 py-2 px-3 bg-sky-600 hover:bg-sky-700 text-white rounded-xl text-xs font-bold active:scale-95 transition-all shadow-xs"
                  >
                    <Share2 className="w-3.5 h-3.5" /> Telegramda ulashish
                  </a>
                </div>
              </div>
            )}

            <button
              onClick={() => setInviteModalOpen(false)}
              className="w-full py-2 bg-slate-800 hover:bg-slate-900 text-white rounded-xl text-xs font-semibold"
            >
              Yopish
            </button>
          </div>
        </div>
      )}

      {/* SUBSTITUTE MODAL */}
      {subModalOpen && currentClass && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl w-full max-w-sm p-4 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150 space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-purple-50 text-purple-700 flex items-center justify-center">
                  <Calendar className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-slate-800">O'rinbosar tayinlash</h3>
                  <p className="text-[10px] text-slate-400">{currentClass.class_name} sinfi uchun</p>
                </div>
              </div>
              <button onClick={() => setSubModalOpen(false)} className="text-slate-400 hover:text-slate-600 p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3">
              <div>
                <label className="block text-[11px] font-medium text-slate-600 mb-1">
                  O'qituvchini tanlang:
                </label>
                <select
                  value={subUserId || ''}
                  onChange={e => setSubUserId(Number(e.target.value))}
                  className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800"
                >
                  {teachers.map(t => (
                    <option key={t.id} value={t.id}>
                      {t.full_name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-[11px] font-medium text-slate-600 mb-1">Qaysi sanadan</label>
                  <input
                    type="date"
                    value={subValidFrom}
                    onChange={e => setSubValidFrom(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2 py-1 text-xs font-semibold text-slate-800"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-slate-600 mb-1">Qaysi sanagacha</label>
                  <input
                    type="date"
                    value={subValidUntil}
                    onChange={e => setSubValidUntil(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2 py-1 text-xs font-semibold text-slate-800"
                  />
                </div>
              </div>

              <p className="text-[10px] text-slate-400">
                O'rinbosar belgilangan muddat davomida ushbu sinf davomatini olish huquqiga ega bo'ladi va eslatmalar o'rinbosarga yuboriladi.
              </p>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                onClick={() => setSubModalOpen(false)}
                className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                Bekor qilish
              </button>
              <button
                onClick={handleSaveSubstitute}
                disabled={savingSub}
                className="px-4 py-2 bg-purple-600 hover:bg-purple-700 active:scale-95 disabled:opacity-50 text-white text-xs font-bold rounded-xl shadow-xs transition-all"
              >
                {savingSub ? "Saqlanmoqda..." : "Biriktirish"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* DIRECT ASSIGN MODAL */}
      {assignModalOpen && currentClass && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl w-full max-w-sm p-4 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150 space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-blue-50 text-blue-700 flex items-center justify-center">
                  <UserPlus className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-slate-800">Sinf rahbari biriktirish</h3>
                  <p className="text-[10px] text-slate-400">{currentClass.class_name} sinfi uchun</p>
                </div>
              </div>
              <button onClick={() => setAssignModalOpen(false)} className="text-slate-400 hover:text-slate-600 p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3">
              <div>
                <label className="block text-[11px] font-medium text-slate-600 mb-1">
                  O'qituvchi Telegram ID raqami:
                </label>
                <input
                  type="number"
                  placeholder="Masalan: 123456789"
                  value={assignTelegramId}
                  onChange={e => setAssignTelegramId(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800"
                />
              </div>

              <div>
                <label className="block text-[11px] font-medium text-slate-600 mb-1">
                  O'qituvchi F.I.Sh (ixtiyoriy):
                </label>
                <input
                  type="text"
                  placeholder="Masalan: Aliyev Vali"
                  value={assignFullName}
                  onChange={e => setAssignFullName(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800"
                />
              </div>

              <p className="text-[10px] text-slate-400">
                O'qituvchi ushbu Telegram ID bilan botga kirganda to'g'ridan-to'g'ri o'z sinfini ko'radi.
              </p>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                onClick={() => setAssignModalOpen(false)}
                className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                Bekor qilish
              </button>
              <button
                onClick={handleSaveDirectAssign}
                disabled={savingAssign || !assignTelegramId}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 active:scale-95 disabled:opacity-50 text-white text-xs font-bold rounded-xl shadow-xs transition-all"
              >
                {savingAssign ? "Saqlanmoqda..." : "Biriktirish"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
