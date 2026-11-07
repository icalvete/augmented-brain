# The panel from another device

[Leer en español](panel-remote.es.md)

The [panel](panel.md) opens on the computer where augmented brain runs, and only there.
To open it from another of your devices, your phone or another computer, augmented
brain uses **[Tailscale](https://tailscale.com)**: a free app that joins your devices in
a private network of their own. Only your devices are in it; nothing is open to the
internet.

```bash
augmented-brain panel remote
```

It starts the panel, publishes it in your Tailscale network, and prints its address,
such as `https://my-laptop.example.ts.net:8443/s/…/today`, and a QR code. Point the
phone's camera at the code, or open the address on the other computer.

If the panel is already open on this computer, it is not stopped: the same panel starts
answering through the network too, and the pages you have open go on working.

## What you need, once

The command checks each of these and, when one is missing, says exactly what to do. It
never installs or changes anything of Tailscale's for you.

1. **Tailscale on both devices**, signed in with the same account:
   [tailscale.com/download](https://tailscale.com/download). It works on Linux, macOS,
   Windows, Android and iPhone.
2. **HTTPS in your Tailscale network**, so the panel travels encrypted with a real
   certificate. Turn it on in Tailscale's admin page,
   [login.tailscale.com/admin/dns](https://login.tailscale.com/admin/dns): **HTTPS
   Certificates → Enable HTTPS**. One thing to know: the names of your machines (such
   as `my-laptop.example.ts.net`) then appear in public logs of certificates. Anyone can
   see that the name exists; nobody outside your devices can reach it.
3. **On Linux, let your user publish.** Tailscale lets only an administrator publish on
   the computer. Once, with your administrator password:

   ```bash
   sudo tailscale set --operator=your-user-name
   ```

   augmented brain prints this line with your name and does not run it. On macOS and
   Windows it is not needed.

## Who can open it

Three things keep it yours:

- **Only your Tailscale network reaches the address.** It does not exist on the
  internet.
- **Only your account gets in.** Tailscale tells the panel who sends each request (the
  account you signed in with, with its own password and second factor), and the panel
  lets in only the owner of the computer. Someone else in your network, or someone you
  share a device with, gets "forbidden" even with the address.
- **The address holds a key** that changes every time the panel starts.

There is no extra password to type: your Tailscale account is the login.

## It is always temporary

The address works while the panel runs. The panel stops by itself after 30 minutes
without use, or now with:

```bash
augmented-brain panel stop
```

When it stops, it takes the address back from Tailscale. If the computer crashed or
restarted with the panel published, the next time a panel starts it takes it back, and
`augmented-brain doctor` warns while it is still there. Port 8443 of your Tailscale
network is the panel's: if you already publish something else there, augmented brain
leaves it alone and says so.

## Check it

`augmented-brain doctor` has a line about the remote panel once you have used it:
whether Tailscale is still installed, connected and with HTTPS, and whether an address
of the panel was left published.
