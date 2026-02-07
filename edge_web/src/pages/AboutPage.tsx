import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { Users, Target, Heart, Zap } from 'lucide-react';

const values = [
  {
    icon: Target,
    title: 'Mission-Driven',
    description: 'We believe computing should be decentralized, accessible, and efficient. Edge Mesh makes it possible.',
  },
  {
    icon: Users,
    title: 'Developer First',
    description: 'Built by developers, for developers. We prioritize developer experience in everything we do.',
  },
  {
    icon: Heart,
    title: 'Open & Transparent',
    description: 'Open-source at heart. We believe in building in public and with the community.',
  },
  {
    icon: Zap,
    title: 'Performance Obsessed',
    description: 'Every millisecond matters. We optimize relentlessly for speed and efficiency.',
  },
];

const team = [
  { name: 'Manav Parikh', avatar: 'MP' },
  { name: 'Chinmay Shringi', avatar: 'CS' },
  { name: 'Bharath Gera', avatar: 'BG' },
  { name: 'Rahil Singhi', avatar: 'RS' },
  { name: 'Sariya Rizwan', avatar: 'SR' },
];

export const AboutPage = () => {
  return (
    <div className="py-24 px-4">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-16"
        >
          <h1 className="text-4xl md:text-5xl font-bold mb-4">
            About <span className="text-primary">Edge Mesh</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            We're building the future of distributed computing, one edge node at a time.
          </p>
        </motion.div>

        {/* Story */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mb-16"
        >
          <GlassCard className="p-8">
            <h2 className="text-2xl font-bold mb-4">Our Story</h2>
            <div className="space-y-4 text-muted-foreground">
              <p>
                Edge Mesh started with a simple observation: the world's computing power is
                increasingly distributed, yet our tools for managing it remain centralized.
              </p>
              <p>
                We set out to build a platform that treats every device—from powerful
                workstations to tiny IoT sensors—as a first-class compute node. Our mesh
                network approach enables seamless workload distribution, real-time monitoring,
                and enterprise-grade security across any device fleet.
              </p>
              <p>
                Today, Edge Mesh powers distributed computing for teams ranging from indie
                developers to Fortune 500 companies. We're just getting started.
              </p>
            </div>
          </GlassCard>
        </motion.div>

        {/* Values */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-16"
        >
          <h2 className="text-2xl font-bold mb-8 text-center">Our Values</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {values.map((value, index) => (
              <motion.div
                key={value.title}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.2 + index * 0.1 }}
              >
                <GlassCard hover className="h-full">
                  <div className="p-4 flex gap-4">
                    <div className="w-12 h-12 rounded-xl bg-primary/20 flex items-center justify-center shrink-0">
                      <value.icon className="w-6 h-6 text-primary" />
                    </div>
                    <div>
                      <h3 className="font-semibold mb-1">{value.title}</h3>
                      <p className="text-sm text-muted-foreground">{value.description}</p>
                    </div>
                  </div>
                </GlassCard>
              </motion.div>
            ))}
          </div>
        </motion.div>

        {/* Team */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          <h2 className="text-2xl font-bold mb-8 text-center">Our Team</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
            {team.map((member, index) => (
              <motion.div
                key={member.name}
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.4 + index * 0.1 }}
                className="text-center"
              >
                <div className="w-20 h-20 mx-auto mb-3 rounded-full bg-gradient-to-br from-primary to-danger-pink flex items-center justify-center text-xl font-bold">
                  {member.avatar}
                </div>
                <h3 className="font-semibold text-sm">{member.name}</h3>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </div>
    </div>
  );
};
