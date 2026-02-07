import { motion } from 'framer-motion';

interface EdgeMeshWordmarkProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  animated?: boolean;
}

const sizeClasses = {
  sm: 'text-lg',
  md: 'text-xl',
  lg: 'text-2xl',
  xl: 'text-4xl',
};

export const EdgeMeshWordmark = ({ size = 'md', animated = false }: EdgeMeshWordmarkProps) => {
  const Wrapper = animated ? motion.div : 'div';
  const animationProps = animated ? {
    initial: { opacity: 0, scale: 0.9 },
    animate: { opacity: 1, scale: 1 },
    transition: { duration: 0.5 }
  } : {};

  return (
    <Wrapper 
      className={`font-orbitron font-bold tracking-wider ${sizeClasses[size]}`}
      {...animationProps}
    >
      <span className="text-primary text-glow-primary">EDGE</span>
      <span className="text-foreground ml-1">MESH</span>
    </Wrapper>
  );
};
