import { motion } from 'framer-motion';
import { EdgeMeshWordmark } from '@/components/EdgeMeshWordmark';

interface WelcomeScreenProps {
  onSuggestionClick: (text: string) => void;
}

const suggestions = [
  'Show my devices',
  'Run status check',
  'Start stream on laptop',
  'Download shared report',
];

export function WelcomeScreen({ onSuggestionClick }: WelcomeScreenProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.98 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.6, ease: 'easeOut' }}
      className="flex flex-col items-center justify-center h-full px-6"
    >
      <div className="flex-1" />

      {/* Logo */}
      <div className="mb-8">
        <EdgeMeshWordmark size="lg" />
      </div>

      {/* Greeting */}
      <h1 className="text-2xl font-bold tracking-tight text-center mb-3">
        <span className="text-foreground">Hello! I am </span>
        <span className="text-safe-green">Edge</span>
        <span className="text-primary">Mesh</span>
        <span className="text-foreground">.</span>
      </h1>

      {/* Subtitle */}
      <p className="text-muted-foreground text-[15px] text-center leading-relaxed font-medium max-w-sm">
        Ask me to run commands, check devices, or manage your network.
      </p>

      {/* Suggestion Chips - 2x2 Grid */}
      <div className="mt-12 w-full max-w-sm">
        <div className="grid grid-cols-2 gap-3">
          {suggestions.map((suggestion) => (
            <button
              key={suggestion}
              onClick={() => onSuggestionClick(suggestion)}
              className="w-full px-3 py-2.5 rounded-full text-[10px] font-medium text-foreground
                         bg-surface-1/45 border border-outline/90
                         hover:bg-surface-variant transition-colors text-center"
            >
              {suggestion}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-[1.5]" />
    </motion.div>
  );
}
