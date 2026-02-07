import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { GlassCard } from '@/components/GlassCard';
import { Check, ArrowRight } from 'lucide-react';

const plans = [
  {
    name: 'Starter',
    price: 'Free',
    description: 'Perfect for personal projects and learning.',
    features: [
      'Up to 5 connected devices',
      'Basic monitoring',
      'Community support',
      '1 GB file transfers/month',
      'Standard encryption',
      '7-day log retention',
    ],
    cta: 'Get Started',
    highlighted: false,
  },
  {
    name: 'Team',
    price: '$49',
    period: '/month',
    description: 'For teams building production applications.',
    features: [
      'Up to 50 connected devices',
      'Advanced monitoring & alerts',
      'Priority email support',
      '50 GB file transfers/month',
      'mTLS encryption',
      '30-day log retention',
      'RBAC access control',
      'Job orchestration',
      'AI assistant access',
    ],
    cta: 'Start Trial',
    highlighted: true,
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    description: 'For organizations with advanced needs.',
    features: [
      'Unlimited devices',
      'Real-time streaming',
      'Dedicated support',
      'Unlimited transfers',
      'Custom security policies',
      '1-year log retention',
      'SSO integration',
      'Custom SLAs',
      'On-premise option',
    ],
    cta: 'Contact Sales',
    highlighted: false,
  },
];

export const PricingPage = () => {
  return (
    <div className="py-24 px-4">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-16"
        >
          <h1 className="text-4xl md:text-5xl font-bold mb-4">
            Simple, Transparent <span className="text-primary">Pricing</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Start free and scale as you grow. No hidden fees.
          </p>
        </motion.div>

        {/* Pricing Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {plans.map((plan, index) => (
            <motion.div
              key={plan.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <GlassCard
                className={`h-full ${
                  plan.highlighted ? 'ring-2 ring-primary glow-primary' : ''
                }`}
              >
                <div className="p-6 space-y-6">
                  {/* Plan header */}
                  <div>
                    {plan.highlighted && (
                      <span className="inline-block px-3 py-1 rounded-full bg-primary/20 text-primary text-xs font-medium mb-4">
                        Most Popular
                      </span>
                    )}
                    <h3 className="text-xl font-bold">{plan.name}</h3>
                    <div className="mt-2">
                      <span className="text-4xl font-bold">{plan.price}</span>
                      {plan.period && (
                        <span className="text-muted-foreground">{plan.period}</span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mt-2">
                      {plan.description}
                    </p>
                  </div>

                  {/* Features */}
                  <ul className="space-y-3">
                    {plan.features.map((feature) => (
                      <li key={feature} className="flex items-start gap-3">
                        <Check className="w-5 h-5 text-safe-green shrink-0 mt-0.5" />
                        <span className="text-sm text-muted-foreground">{feature}</span>
                      </li>
                    ))}
                  </ul>

                  {/* CTA */}
                  <Button
                    asChild
                    className={`w-full ${
                      plan.highlighted
                        ? 'bg-primary hover:bg-primary/90'
                        : 'bg-surface-variant hover:bg-surface-variant/80'
                    }`}
                  >
                    <Link to="/console/connect">
                      {plan.cta}
                      <ArrowRight className="ml-2 w-4 h-4" />
                    </Link>
                  </Button>
                </div>
              </GlassCard>
            </motion.div>
          ))}
        </div>

        {/* FAQ teaser */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.5 }}
          className="text-center mt-16"
        >
          <p className="text-muted-foreground">
            Questions? <a href="#" className="text-primary hover:underline">Contact our team</a>
          </p>
        </motion.div>
      </div>
    </div>
  );
};
