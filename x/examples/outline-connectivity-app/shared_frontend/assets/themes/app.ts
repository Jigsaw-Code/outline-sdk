import { createTheme } from '@vanilla-extract/css';
import { Theme as JigsawTheme } from "./jigsaw";

export const [Class, Theme] = createTheme({
  font: {
    sansSerif: 'Jigsaw Sans, Helvetica Neue, Helvetica, Arial, sans-serif',
    monospace: 'Menlo, Courier, monospace',
  },
  size: {
    border: '0.5px',
    borderSwitch: '2px',
    cornerRadius: '0.2rem',
    gapInner: '0.25rem',
    gapNarrow: '0.5rem',
    gap: '1rem',
    fontSmall: '0.75rem',
    font: '1rem',
    fontLarge: '1.85rem',
    switch: '1.5rem',
    appWidthMin: '320px',
    appWidthMax: '560px',
  },
  timing: {
    snappy: '140ms',
  },
  color: {
    text: JigsawTheme.color.black,
    textBrand: JigsawTheme.color.white,
    textHighlight: JigsawTheme.color.white,
    textMuted: JigsawTheme.color.gray,
    background: JigsawTheme.color.white,
    backgroundBrand: JigsawTheme.color.green,
    backgroundHighlight: 'hsl(156, 33%, 55%)',
    backgroundMuted: JigsawTheme.color.grayMedium,
    successText: JigsawTheme.color.greenMedium,
    successBackground: JigsawTheme.color.greenLight,
    errorText: JigsawTheme.color.red,
    errorBackground: JigsawTheme.color.grayLight,
  },
});
