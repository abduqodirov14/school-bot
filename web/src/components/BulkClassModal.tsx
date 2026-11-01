import React, { useState } from 'react';
import { api } from '../api/client';
import { X, Layers, Check } from 'lucide-react';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (msg: string) => void;
}

const COMMON_PARALLELS = ['A', 'B', 'V', 'G', 'D', 'E', 'J'];

export const BulkClassModal: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
  const [startGrade, setStartGrade] = useState<number>(5);
  const [endGrade, setEndGrade] = useState<number>(11);
  const [selectedParallels, setSelectedParallels] = useState<string[]>(['A', 'B', 'V', 'G']);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const toggleParallel = (p: string) => {
    setSelectedParallels(prev =>
      prev.includes(p) ? prev.filter(x => x !== p) : [...prev, p]
    );
  };

  const handleBulkCreate = async () => {
    if (startGrade > endGrade) {
      setError("Boshlang'ich sinf yakuniy sinfdan kichik yoki teng bo'lishi kerak");
      return;
    }
    if (selectedParallels.length === 0) {
      setError("Kamida bitta parallel (harf) tanlang");
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      const res = await api.bulkCreateClasses({
        start_grade: startGrade,
        end_grade: endGrade,
        parallels: selectedParallels,
      });

      onSuccess(
        `${res.created_count} ta sinf yaratildi (${res.skipped_count} ta mavjud sinf o'tkazib yuborildi)`
      );
      onClose();
    } catch (err: any) {
      setError(err.message || "Sinflarni yaratishda xatolik");
    } finally {
      setSubmitting(false);
    }
  };

  const totalPossible = Math.max(0, endGrade - startGrade + 1) * selectedParallels.length;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl w-full max-w-sm p-4 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150">
        <div className="flex items-center justify-between pb-3 border-b border-slate-100">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-700 flex items-center justify-center">
              <Layers className="w-4 h-4" />
            </div>
            <h3 className="text-sm font-bold text-slate-800">Ommaviy sinflar yaratish</h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600 p-1">
            <X className="w-4 h-4" />
          </button>
        </div>

        {error && (
          <div className="mt-3 p-2.5 bg-rose-50 text-rose-700 text-xs rounded-xl border border-rose-200">
            {error}
          </div>
        )}

        <div className="space-y-4 py-4">
          {/* Grade Range */}
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-[11px] font-medium text-slate-600 mb-1">Qaysi sinfdan</label>
              <select
                value={startGrade}
                onChange={e => setStartGrade(Number(e.target.value))}
                className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800"
              >
                {Array.from({ length: 11 }, (_, i) => i + 1).map(g => (
                  <option key={g} value={g}>{g}-sinf</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-[11px] font-medium text-slate-600 mb-1">Qaysi sinfgacha</label>
              <select
                value={endGrade}
                onChange={e => setEndGrade(Number(e.target.value))}
                className="w-full bg-slate-50 border border-slate-300 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-800"
              >
                {Array.from({ length: 11 }, (_, i) => i + 1).map(g => (
                  <option key={g} value={g}>{g}-sinf</option>
                ))}
              </select>
            </div>
          </div>

          {/* Parallels */}
          <div>
            <label className="block text-[11px] font-medium text-slate-600 mb-1.5">
              Parallellar (harflar):
            </label>
            <div className="flex flex-wrap gap-1.5">
              {COMMON_PARALLELS.map(p => {
                const active = selectedParallels.includes(p);
                return (
                  <button
                    key={p}
                    type="button"
                    onClick={() => toggleParallel(p)}
                    className={`w-9 h-9 rounded-xl text-xs font-bold transition-all flex items-center justify-center gap-0.5 border ${
                      active
                        ? 'bg-emerald-600 text-white border-emerald-600 shadow-xs scale-102'
                        : 'bg-slate-50 text-slate-600 border-slate-200 hover:bg-slate-100'
                    }`}
                  >
                    {p}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="bg-slate-50 rounded-xl p-3 border border-slate-100 text-xs text-slate-500">
            Tanlangan parametrlar bo'yicha jami: <strong>{totalPossible} ta</strong> sinf ko'rib chiqiladi. Mavjud bo'lganlari o'tkazib yuboriladi.
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg"
          >
            Bekor qilish
          </button>
          <button
            onClick={handleBulkCreate}
            disabled={submitting || totalPossible === 0}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 active:scale-95 disabled:opacity-50 text-white text-xs font-bold rounded-xl shadow-xs transition-all flex items-center gap-1.5"
          >
            <Check className="w-3.5 h-3.5 stroke-[3]" />
            {submitting ? "Yaratilmoqda..." : "Barchasini yaratish"}
          </button>
        </div>
      </div>
    </div>
  );
};
