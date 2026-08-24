import { useState, useEffect } from 'react';

export function useFlag(flag: { isEnabled: () => boolean }): boolean {
  const [value, setValue] = useState(() => flag.isEnabled());

  useEffect(() => {
    const interval = setInterval(() => {
      const current = flag.isEnabled();
      setValue((prev) => (prev !== current ? current : prev));
    }, 5000);
    return () => clearInterval(interval);
  }, [flag]);

  return value;
}
