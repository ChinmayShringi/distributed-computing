import { Outlet } from 'react-router-dom';
import { PublicNavbar } from '@/components/PublicNavbar';
import { Footer } from '@/components/Footer';

export const PublicLayout = () => {
  return (
    <div className="min-h-screen bg-background bg-noise">
      <PublicNavbar />
      <main className="pt-16">
        <Outlet />
      </main>
      <Footer />
    </div>
  );
};
