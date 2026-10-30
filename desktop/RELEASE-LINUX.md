# Margo Desktop for Arch Linux

This experimental portable folder contains the Margo desktop app and the Go
engine built from the same release tag. Keep `margo-desktop` and `margo-engine`
in the same directory.

On Arch Linux and Omarchy, install the runtime dependencies:

```sh
sudo pacman -S --needed webkit2gtk-4.1
```

For slide thumbnail capture and PDF export, also install `chromium`. Extract the
archive and launch `margo-desktop` from the extracted directory. No separately
installed Margo CLI is required. The app does not remove or modify your projects
when it is removed.

This archive is not a signed system package and does not install a desktop entry.
It is an experimental release build; native Omarchy/Hyprland acceptance is still
required.
