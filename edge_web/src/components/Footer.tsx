import { Link } from 'react-router-dom';
import { EdgeMeshWordmark } from '@/components/EdgeMeshWordmark';
import { Github, Twitter, Linkedin } from 'lucide-react';

export const Footer = () => {
  return (
    <footer className="border-t border-outline bg-surface-1">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="py-8">
          <p className="text-center text-sm text-muted-foreground">
            © 2026 Edge Mesh. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
};
