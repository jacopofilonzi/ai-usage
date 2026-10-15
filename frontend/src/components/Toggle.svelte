<script lang="ts">
  // On/off switch; `label` is the accessible name (the visible label sits next to it).
  interface Props {
    checked: boolean;
    label: string;
    disabled?: boolean;
    onchange: (checked: boolean) => void;
  }

  let { checked, label, disabled = false, onchange }: Props = $props();
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  aria-label={label}
  {disabled}
  class="toggle"
  class:on={checked}
  onclick={() => onchange(!checked)}
>
  <span class="knob"></span>
</button>

<style>
  .toggle {
    position: relative;
    flex: none;
    width: 2.5rem;
    height: 1.5rem;
    border-radius: 999px;
    background: var(--c-border);
    transition: background-color 150ms;
  }

  .toggle:focus-visible {
    outline: 2px solid var(--c-accent);
    outline-offset: 2px;
  }

  .toggle:disabled {
    opacity: 0.5;
  }

  .toggle.on {
    background: var(--c-accent);
  }

  .knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 1.125rem;
    height: 1.125rem;
    border-radius: 999px;
    background: #fff;
    box-shadow: 0 1px 2px rgb(0 0 0 / 0.25);
    transition: transform 150ms ease-out;
  }

  .on .knob {
    transform: translateX(1rem);
  }
</style>
