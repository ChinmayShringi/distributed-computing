import { motion } from 'framer-motion';

export function ThinkingIndicator() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="flex items-center gap-1 py-4"
    >
      {[0, 1, 2].map((index) => (
        <motion.div
          key={index}
          className="w-2 h-2 rounded-full bg-muted-foreground"
          animate={{
            scale: [0.8, 1.2, 0.8],
          }}
          transition={{
            duration: 0.6,
            repeat: Infinity,
            delay: index * 0.2,
            ease: 'easeInOut',
          }}
        />
      ))}
    </motion.div>
  );
}
