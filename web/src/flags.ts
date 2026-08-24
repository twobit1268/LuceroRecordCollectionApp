import Rox from 'rox-browser';

export const Flags = {
  dark_mode: new Rox.Flag(false),
  grid_view: new Rox.Flag(false),
  export_csv: new Rox.Flag(false),
};

export async function initFlags() {
  const key = import.meta.env.VITE_CLOUDBEES_SDK_KEY;
  if (!key) return;
  try {
    Rox.register('', Flags);
    await Rox.setup(key);
  } catch (e) {
    console.warn('CloudBees flag init failed, using defaults', e);
  }
}
