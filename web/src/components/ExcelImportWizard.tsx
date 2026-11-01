import React, { useState, useEffect, useRef } from 'react';
import { api, getInitData } from '../api/client';
import type { ImportJobStatusResponse, ImportPreviewResponse } from '../types';
import {
  X,
  FileSpreadsheet,
  Download,
  AlertTriangle,
  CheckCircle2,
  AlertCircle,
  ArrowRight,
  RefreshCw,
  UserX,
} from 'lucide-react';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (msg: string) => void;
}

export const ExcelImportWizard: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
  const [step, setStep] = useState<'upload' | 'preview' | 'importing' | 'done'>('upload');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);

  // Preview state
  const [loadingPreview, setLoadingPreview] = useState<boolean>(false);
  const [preview, setPreview] = useState<ImportPreviewResponse | null>(null);
  const [createMissingClasses, setCreateMissingClasses] = useState<boolean>(true);

  // Running & status state
  const [jobId, setJobId] = useState<string | null>(null);
  const [jobStatus, setJobStatus] = useState<ImportJobStatusResponse | null>(null);
  const [isDeactivatingMissing, setIsDeactivatingMissing] = useState<boolean>(false);
  const [deactivatedMsg, setDeactivatedMsg] = useState<string | null>(null);

  const pollTimerRef = useRef<any>(null);

  // Reset when opened
  useEffect(() => {
    if (isOpen) {
      setStep('upload');
      setSelectedFile(null);
      setFileError(null);
      setPreview(null);
      setJobId(null);
      setJobStatus(null);
      setCreateMissingClasses(true);
      setDeactivatedMsg(null);
    }
    return () => {
      if (pollTimerRef.current) clearInterval(pollTimerRef.current);
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    setFileError(null);
    if (!file) {
      setSelectedFile(null);
      return;
    }

    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      setFileError("Faqat .xlsx formatidagi Excel fayllar qabul qilinadi");
      setSelectedFile(null);
      return;
    }

    if (file.size > 5 * 1024 * 1024) {
      setFileError("Fayl hajmi 5 MB dan oshmasligi kerak");
      setSelectedFile(null);
      return;
    }

    setSelectedFile(file);
  };

  const handleDownloadTemplate = async () => {
    try {
      const initData = getInitData();
      const headers = new Headers();
      if (initData) headers.set('X-Telegram-Init-Data', initData);

      const resp = await fetch('/api/import/template', { headers });
      if (!resp.ok) throw new Error("Shablonni yuklab bo'lmadi");

      const blob = await resp.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = "Maktab_Oquvchilar_Import_Shablon.xlsx";
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      setFileError(err.message || "Shablonni yuklab olishda xatolik");
    }
  };

  const handlePreview = async () => {
    if (!selectedFile) return;
    setLoadingPreview(true);
    setFileError(null);
    try {
      const res = await api.previewImport(selectedFile);
      setPreview(res);
      setJobId(res.job_id);
      setStep('preview');
    } catch (err: any) {
      setFileError(err.message || "Excel faylni o'qishda xatolik");
    } finally {
      setLoadingPreview(false);
    }
  };

  const handleStartCommit = async () => {
    if (!jobId) return;
    try {
      await api.commitImport(jobId, createMissingClasses);
      setStep('importing');
      startPolling(jobId);
    } catch (err: any) {
      setFileError(err.message || "Importni boshlashda xatolik");
    }
  };

  const startPolling = (id: string) => {
    if (pollTimerRef.current) clearInterval(pollTimerRef.current);

    pollTimerRef.current = setInterval(async () => {
      try {
        const status = await api.getImportJob(id);
        setJobStatus(status);

        if (status.status === 'done' || status.status === 'failed') {
          clearInterval(pollTimerRef.current);
          setStep('done');
        }
      } catch {
        // continue polling
      }
    }, 1000);
  };

  const handleDownloadErrorsReport = async () => {
    if (!jobId) return;
    try {
      const initData = getInitData();
      const headers = new Headers();
      if (initData) headers.set('X-Telegram-Init-Data', initData);

      const resp = await fetch(`/api/import/${jobId}/errors/download`, { headers });
      if (!resp.ok) throw new Error("Hisobotni yuklab bo'lmadi");

      const blob = await resp.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `Import_Xatoliklar_${jobId}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      alert(err.message || "Yuklab olishda xatolik");
    }
  };

  const handleDeactivateMissing = async () => {
    if (!jobStatus?.summary?.missing_from_file_ids?.length) return;
    if (!window.confirm("Rostdan ham faylda mavjud bo'lmagan o'quvchilarni nofaol qilmoqchimisiz?")) return;

    setIsDeactivatingMissing(true);
    try {
      const res = await api.deactivateMissingStudents(jobStatus.summary.missing_from_file_ids);
      setDeactivatedMsg(`${res.deactivated_count} nafar o'quvchi muvaffaqiyatli nofaol qilindi.`);
    } catch (err: any) {
      alert(err.message || "Nofaol qilishda xatolik");
    } finally {
      setIsDeactivatingMissing(false);
    }
  };

  const progressPercent = jobStatus?.total_rows
    ? Math.min(100, Math.round((jobStatus.processed_rows / jobStatus.total_rows) * 100))
    : 0;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-3">
      <div className="bg-white rounded-2xl w-full max-w-lg p-4 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150 max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-slate-100 shrink-0">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-700 flex items-center justify-center">
              <FileSpreadsheet className="w-4 h-4" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-slate-800">O'quvchilarni Excel orqali import qilish</h3>
              <p className="text-[10px] text-slate-400">
                {step === 'upload' && "1-qadam: Faylni tanlash"}
                {step === 'preview' && "2-qadam: Ma'lumotlarni tekshirish"}
                {step === 'importing' && "3-qadam: Import jarayoni"}
                {step === 'done' && "Yakunlandi"}
              </p>
            </div>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600 p-1">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content Body */}
        <div className="overflow-y-auto py-3 space-y-3 flex-1 pr-1 text-slate-700">
          {fileError && (
            <div className="p-2.5 bg-rose-50 text-rose-700 text-xs rounded-xl border border-rose-200 flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{fileError}</span>
            </div>
          )}

          {/* STEP 1: UPLOAD */}
          {step === 'upload' && (
            <div className="space-y-4">
              <div className="p-3 bg-slate-50 border border-slate-200 rounded-xl space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-slate-700">Namuna shablon</span>
                  <button
                    onClick={handleDownloadTemplate}
                    className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold text-emerald-700 bg-emerald-50 hover:bg-emerald-100 rounded-lg border border-emerald-200 transition-all"
                  >
                    <Download className="w-3.5 h-3.5" /> Shablonni yuklash (.xlsx)
                  </button>
                </div>
                <p className="text-[11px] text-slate-500">
                  Ustunlar nomi: <strong>Sinf</strong> va <strong>F.I.Sh</strong> (yoki <strong>Familiya</strong> va <strong>Ism</strong>).
                  Bitta sahifada yoki har bir sinf uchun alohida sahifada kiritilishi mumkin.
                </p>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1.5">
                  Excel (.xlsx) faylini tanlang:
                </label>
                <input
                  type="file"
                  accept=".xlsx"
                  onChange={handleFileChange}
                  className="block w-full text-xs text-slate-600 file:mr-3 file:py-2 file:px-3 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-emerald-50 file:text-emerald-700 hover:file:bg-emerald-100 file:cursor-pointer border border-slate-200 rounded-xl p-1 bg-slate-50"
                />
                <p className="text-[10px] text-slate-400 mt-1">Maksimal hajm: 5 MB (10,000 qatorgacha)</p>
              </div>

              {selectedFile && (
                <div className="p-3 bg-emerald-50/50 border border-emerald-200 rounded-xl text-xs flex items-center justify-between">
                  <span className="font-semibold text-emerald-900 truncate">{selectedFile.name}</span>
                  <span className="text-[10px] text-emerald-700 shrink-0 ml-2">
                    {(selectedFile.size / 1024).toFixed(1)} KB
                  </span>
                </div>
              )}
            </div>
          )}

          {/* STEP 2: PREVIEW */}
          {step === 'preview' && preview && (
            <div className="space-y-3">
              {/* Summary Cards Grid */}
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="bg-slate-50 p-2.5 rounded-xl border border-slate-200">
                  <span className="text-slate-500 text-[11px]">Jami o'quvchilar:</span>
                  <p className="text-base font-black text-slate-900">{preview.total_rows}</p>
                </div>
                <div className="bg-emerald-50 p-2.5 rounded-xl border border-emerald-200">
                  <span className="text-emerald-700 text-[11px]">Yangi o'quvchilar:</span>
                  <p className="text-base font-black text-emerald-700">{preview.new_students}</p>
                </div>
                <div className="bg-blue-50 p-2.5 rounded-xl border border-blue-200">
                  <span className="text-blue-700 text-[11px]">Mavjud o'quvchilar:</span>
                  <p className="text-base font-black text-blue-700">{preview.existing_students}</p>
                </div>
                <div className="bg-amber-50 p-2.5 rounded-xl border border-amber-200">
                  <span className="text-amber-700 text-[11px]">Takroriy / Xatolar:</span>
                  <p className="text-base font-black text-amber-700">
                    {preview.duplicate_rows + preview.error_rows}
                  </p>
                </div>
              </div>

              {/* Missing classes warning */}
              {preview.missing_classes && preview.missing_classes.length > 0 && (
                <div className="p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs space-y-2">
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                    <div>
                      <p className="font-bold text-amber-900">
                        {preview.missing_classes.length} ta yangi sinf aniqlandi:
                      </p>
                      <p className="text-[11px] text-amber-800 mt-0.5">
                        {preview.missing_classes.join(', ')}
                      </p>
                    </div>
                  </div>
                  <label className="flex items-center gap-2 pt-1 text-[11px] font-semibold text-amber-950 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={createMissingClasses}
                      onChange={e => setCreateMissingClasses(e.target.checked)}
                      className="rounded border-amber-400 text-emerald-600 focus:ring-emerald-500 w-4 h-4"
                    />
                    Ushbu yangi sinflarni avtomatik tarzda yaratish
                  </label>
                </div>
              )}

              {/* Errors table (first 50) */}
              {preview.errors && preview.errors.length > 0 && (
                <div className="space-y-1">
                  <p className="text-xs font-bold text-rose-700">
                    Aniqlangan xatoliklar ({preview.errors.length} ta):
                  </p>
                  <div className="max-h-40 overflow-y-auto border border-rose-200 rounded-xl">
                    <table className="w-full text-left text-[11px]">
                      <thead className="bg-rose-50 text-rose-900 sticky top-0">
                        <tr>
                          <th className="p-1.5 font-bold">Qator</th>
                          <th className="p-1.5 font-bold">Sahifa</th>
                          <th className="p-1.5 font-bold">Sabab</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-rose-100">
                        {preview.errors.slice(0, 50).map((err, idx) => (
                          <tr key={idx} className="hover:bg-rose-50/50">
                            <td className="p-1.5 font-mono">{err.row}</td>
                            <td className="p-1.5">{err.sheet || 'Asosiy'}</td>
                            <td className="p-1.5 text-rose-700">{err.reason}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* STEP 3: IMPORTING PROGRESS */}
          {step === 'importing' && (
            <div className="py-8 text-center space-y-4">
              <RefreshCw className="w-8 h-8 text-emerald-600 animate-spin mx-auto" />
              <div>
                <h4 className="text-sm font-bold text-slate-800">Ma'lumotlar bazasiga yozilmoqda...</h4>
                <p className="text-xs text-slate-500 mt-1">
                  {jobStatus
                    ? `${jobStatus.processed_rows} / ${jobStatus.total_rows} ta qator qayta ishlandi`
                    : "Jarayon boshlanmoqda..."}
                </p>
              </div>

              {/* Progress Bar */}
              <div className="w-full bg-slate-100 rounded-full h-3 overflow-hidden border border-slate-200">
                <div
                  className="bg-emerald-600 h-full transition-all duration-300"
                  style={{ width: `${progressPercent}%` }}
                />
              </div>
              <span className="text-xs font-bold text-emerald-700">{progressPercent}%</span>
            </div>
          )}

          {/* STEP 4: DONE */}
          {step === 'done' && jobStatus && (
            <div className="space-y-4">
              <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-2xl text-center space-y-2">
                <CheckCircle2 className="w-10 h-10 text-emerald-600 mx-auto" />
                <h4 className="text-base font-bold text-emerald-900">Import muvaffaqiyatli yakunlandi!</h4>
                <p className="text-xs text-emerald-700">
                  {jobStatus.created_count} ta yangi o'quvchi qo'shildi, {jobStatus.updated_count} tasi yangilandi.
                </p>
              </div>

              {/* Stats Grid */}
              <div className="grid grid-cols-4 gap-2 text-center text-xs">
                <div className="p-2 bg-slate-50 rounded-xl border border-slate-200">
                  <span className="block font-bold text-slate-900 text-sm">{jobStatus.created_count}</span>
                  <span className="text-[10px] text-slate-500">Yaratildi</span>
                </div>
                <div className="p-2 bg-slate-50 rounded-xl border border-slate-200">
                  <span className="block font-bold text-slate-900 text-sm">{jobStatus.updated_count}</span>
                  <span className="text-[10px] text-slate-500">Yangilandi</span>
                </div>
                <div className="p-2 bg-slate-50 rounded-xl border border-slate-200">
                  <span className="block font-bold text-slate-900 text-sm">{jobStatus.skipped_count}</span>
                  <span className="text-[10px] text-slate-500">O'tkazildi</span>
                </div>
                <div className="p-2 bg-slate-50 rounded-xl border border-slate-200">
                  <span className="block font-bold text-rose-600 text-sm">{jobStatus.failed_count}</span>
                  <span className="text-[10px] text-rose-500">Xatolar</span>
                </div>
              </div>

              {/* Error report button if any failed */}
              {jobStatus.failed_count > 0 && (
                <button
                  onClick={handleDownloadErrorsReport}
                  className="w-full py-2 px-3 bg-rose-50 hover:bg-rose-100 text-rose-700 rounded-xl text-xs font-semibold border border-rose-200 flex items-center justify-center gap-1.5 transition-all"
                >
                  <Download className="w-3.5 h-3.5" /> Xatoliklar hisobotini yuklab olish (.xlsx)
                </button>
              )}

              {/* Missing from file notice */}
              {jobStatus.missing_from_file_count > 0 && (
                <div className="p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="font-bold text-amber-900">
                      {jobStatus.missing_from_file_count} nafar o'quvchi faylda yo'q
                    </span>
                    <button
                      onClick={handleDeactivateMissing}
                      disabled={isDeactivatingMissing}
                      className="px-2.5 py-1 bg-amber-600 hover:bg-amber-700 text-white rounded-lg text-[11px] font-bold shadow-xs active:scale-95 transition-all flex items-center gap-1"
                    >
                      <UserX className="w-3 h-3" />
                      {isDeactivatingMissing ? "Bajarilmoqda..." : "Nofaol qilish"}
                    </button>
                  </div>
                  {deactivatedMsg && (
                    <p className="text-[11px] text-emerald-700 font-semibold">{deactivatedMsg}</p>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer Actions */}
        <div className="flex items-center justify-between pt-3 border-t border-slate-100 shrink-0">
          {step === 'upload' && (
            <>
              <button
                onClick={onClose}
                className="px-3 py-1.5 text-xs font-semibold text-slate-500 hover:bg-slate-100 rounded-lg"
              >
                Bekor qilish
              </button>
              <button
                onClick={handlePreview}
                disabled={!selectedFile || loadingPreview}
                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 active:scale-95 disabled:opacity-50 text-white text-xs font-bold rounded-xl shadow-xs transition-all flex items-center gap-1.5"
              >
                {loadingPreview ? (
                  <>
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" /> O'qilmoqda...
                  </>
                ) : (
                  <>
                    Keyingisi <ArrowRight className="w-3.5 h-3.5" />
                  </>
                )}
              </button>
            </>
          )}

          {step === 'preview' && (
            <>
              <button
                onClick={() => setStep('upload')}
                className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                Orqaga
              </button>
              <button
                onClick={handleStartCommit}
                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 active:scale-95 text-white text-xs font-bold rounded-xl shadow-xs transition-all flex items-center gap-1.5"
              >
                Tasdiqlash va import qilish
              </button>
            </>
          )}

          {step === 'importing' && (
            <div className="w-full text-center text-xs text-slate-400">
              Iltimos, kuting. Sahifani yopmang...
            </div>
          )}

          {step === 'done' && (
            <button
              onClick={() => {
                onSuccess("Excel import muvaffaqiyatli yakunlandi!");
                onClose();
              }}
              className="w-full py-2 bg-slate-800 hover:bg-slate-900 active:scale-95 text-white text-xs font-bold rounded-xl shadow-xs transition-all"
            >
              Yopish
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
