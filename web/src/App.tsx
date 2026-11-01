import { useEffect, useState } from 'react';
import { api, getInitData } from './api/client';
import type { UserProfileResponse } from './types';
import { TeacherAttendance } from './components/TeacherAttendance';
import { DirectorDashboard } from './components/DirectorDashboard';
import { AlertTriangle, Clock, School } from 'lucide-react';

export function App() {
  const [profile, setProfile] = useState<UserProfileResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Notify Telegram Mini App is ready
    if (typeof window !== 'undefined' && window.Telegram?.WebApp) {
      window.Telegram.WebApp.ready();
      window.Telegram.WebApp.expand();
      window.Telegram.WebApp.setHeaderColor?.('#ffffff');
      window.Telegram.WebApp.setBackgroundColor?.('#f8fafc');
    }

    loadProfile();
  }, []);

  const loadProfile = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getMe();
      setProfile(data);
    } catch (err: any) {
      // If initData is missing (e.g. local browser preview outside Telegram)
      if (!getInitData()) {
        setError("Ilovani Telegram boti orqali oching (/start -> Mini App tugmasi).");
      } else {
        setError(err.message || "Foydalanuvchi ma'lumotlarini yuklab bo'lmadi");
      }
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-4 text-center">
        <div className="w-8 h-8 border-3 border-emerald-600 border-t-transparent rounded-full animate-spin mb-3" />
        <p className="text-xs font-semibold text-slate-600">Maktab tizimi yuklanmoqda...</p>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-6 text-center">
        <div className="w-12 h-12 bg-amber-100 text-amber-700 rounded-2xl flex items-center justify-center mb-4">
          <AlertTriangle className="w-6 h-6" />
        </div>
        <h2 className="text-base font-bold text-slate-900 mb-1">Kirish talab etiladi</h2>
        <p className="text-xs text-slate-500 max-w-xs mb-6">{error}</p>
        <button
          onClick={loadProfile}
          className="px-4 py-2 bg-slate-800 text-white rounded-xl text-xs font-semibold hover:bg-slate-900 active:scale-95 transition-all"
        >
          Qayta urinish
        </button>
      </div>
    );
  }

  // Pending user state
  if (profile.user.status === 'pending') {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-6 text-center">
        <div className="w-12 h-12 bg-amber-100 text-amber-600 rounded-2xl flex items-center justify-center mb-4">
          <Clock className="w-6 h-6" />
        </div>
        <h2 className="text-base font-bold text-slate-900 mb-1">Arizangiz ko'rib chiqilmoqda</h2>
        <p className="text-xs text-slate-500 max-w-xs">
          Hurmatli {profile.user.full_name}, maktab direktori hisobingizni tasdiqlagach, tizimdan foydalanishingiz mumkin bo'ladi.
        </p>
      </div>
    );
  }

  // Teacher Screen
  if (profile.user.role === 'teacher') {
    if (!profile.classes || profile.classes.length === 0) {
      return (
        <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-6 text-center">
          <div className="w-12 h-12 bg-slate-200 text-slate-600 rounded-2xl flex items-center justify-center mb-4">
            <School className="w-6 h-6" />
          </div>
          <h2 className="text-base font-bold text-slate-900 mb-1">Sinf biriktirilmagan</h2>
          <p className="text-xs text-slate-500 max-w-xs">
            Sizga hali hech qanday sinf biriktirilmagan. Iltimos, maktab direktori bilan bog'laning.
          </p>
        </div>
      );
    }

    return (
      <TeacherAttendance
        initialClasses={profile.classes}
        schoolName={profile.school?.name || "Maktab Davomati"}
      />
    );
  }

  // Director Screen
  return (
    <DirectorDashboard
      schoolName={profile.school?.name || "Maktab Davomati"}
    />
  );
}

export default App;
