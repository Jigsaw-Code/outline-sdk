import { style } from "@vanilla-extract/css";
import { App } from "../../../../assets/themes";

export const Main = style({
  display: "block",
  gap: App.Theme.size.gapNarrow,
  margin: "0 auto",
  maxWidth: App.Theme.size.appWidthMax,
  padding: App.Theme.size.gap,
  paddingBottom: `calc(${App.Theme.size.gap} * 6)`,
  marginTop: App.Theme.size.gap,
});

const field = {
  display: "block",
  marginBottom: App.Theme.size.gap,
};

export const Field = style(field);
export const FieldGroup = style({
  ...field,
  background: App.Theme.color.backgroundMuted,
  borderRadius: App.Theme.size.cornerRadius,
  padding: App.Theme.size.gap,
  display: "flex",
  justifyContent: "space-around",
  "@media": {
    "(prefers-contrast: more)": {
      border: `${App.Theme.size.border} solid ${App.Theme.color.text}`,
      background: App.Theme.color.background
    }
  }
});

export const FieldGroupItem = style({
  alignItems: "center",
  display: "flex",
  gap: App.Theme.size.gapNarrow,
});

export const FieldHeader = style({
  alignItems: "center",
  display: "flex",
  gap: App.Theme.size.gapNarrow,
  marginBottom: App.Theme.size.gapNarrow,
});

export const FieldHeaderLabel = style({
  cursor: "pointer",
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.font,
  color: App.Theme.color.text,
  fontWeight: "bold",
});

export const FieldHeaderInformation = style({
  opacity: 0.5,
  fontSize: App.Theme.size.fontSmall,
  cursor: "help",
  ":hover": {
    opacity: 1,
  },
  "@media": {
    "(prefers-contrast: more)": {
      opacity: 1
    },
    [`only screen and (min-width: ${App.Theme.size.appWidthMin})`]: {
      display: "none"
    }  
  }
});

export const FieldHeaderRequired = style({
  fontWeight: "bold",
  color: "red",
});

export const FieldLabel = style({
  cursor: "pointer",
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.font,
  color: App.Theme.color.text,
});

export const FieldInput = style({
  borderRadius: App.Theme.size.cornerRadius,
  border: App.Theme.size.border,
  display: "block",
  fontFamily: App.Theme.font.monospace,
  padding: App.Theme.size.gapNarrow,
  width: "100%",
  background: App.Theme.color.backgroundMuted,
  color: App.Theme.color.text,
  ":focus": {
    borderColor: App.Theme.color.backgroundHighlight,
  },
  "@media": {
    "(prefers-contrast: more)": {
      background: App.Theme.color.background
    }
  }
});

export const FieldTextInput = FieldInput;

export const FieldCheckbox = style({
  background: App.Theme.color.background,
  borderRadius: App.Theme.size.font,
  border: App.Theme.size.borderSwitch,
  cursor: "pointer",
  display: "inline-block",
  flexShrink: 0,
  height: App.Theme.size.font,
  position: "relative",
  transition: `background ${App.Theme.timing.snappy} ease-in-out, border ${App.Theme.timing.snappy} ease-in-out`,
  width: App.Theme.size.switch,
  ":checked": {
    background: App.Theme.color.backgroundHighlight,
    borderColor: App.Theme.color.backgroundHighlight,
  },
  "::after": {
    background: App.Theme.color.text,
    borderRadius: App.Theme.size.fontSmall,
    color: "rgba(0, 0, 0, 0%)",
    content: "🤪",
    display: "inline-block",
    height: App.Theme.size.fontSmall,
    left: 0,
    position: "absolute",
    top: "50%",
    transform: "translate(0, -50%)",
    transition: `transform ${App.Theme.timing.snappy} ease-in-out, left ${App.Theme.timing.snappy} ease-in-out`,
    width: App.Theme.size.fontSmall,
    willChange: "transform, left",
  },
  selectors: {
    "&::after:checked": {
      left: "100%",
      transform: "translate(-100%, -50%)",
    },
  },
  "@media": {
    "(prefers-reduced-motion: reduce)": {
      transition: "none",
      "::after": {
        transition: "none"
      }
    },
    "(prefers-contrast: more)": {
      borderColor: App.Theme.color.text,
      ":checked": {
        background: App.Theme.color.background,
        borderColor: App.Theme.color.text
      }
    }
  }
});

const submitInteraction = {
  background: App.Theme.color.backgroundHighlight,
  color: App.Theme.color.textHighlight,
};

export const Submit = style({
  background: App.Theme.color.text,
  borderRadius: App.Theme.size.cornerRadius,
  color: App.Theme.color.background,
  cursor: "pointer",
  fontFamily: App.Theme.font.sansSerif,
  padding: App.Theme.size.gapNarrow,
  textAlign: "center",
  ":focus": submitInteraction,
  ":hover": submitInteraction,
  ":disabled": {
    background: App.Theme.color.backgroundMuted,
    color: App.Theme.color.textMuted,
    cursor: "not-allowed",
  }
});
