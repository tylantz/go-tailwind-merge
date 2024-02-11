Download the standalone cli

```
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
chmod +x tailwindcss-macos-arm64
mv tailwindcss-macos-arm64 tailwindcss
```

Generate the initial files

```
./tailwindcss init
```

Create an input.css with the @tailwind options

To generate all utility classes https://github.com/tailwindlabs/tailwindcss/discussions/10379#discussioncomment-7987635,
make the safelist edit below

```
 module.exports = {
   // (your config props)
+  safelist: [
+    {
+      pattern: /.+/,
+    },
+  ],
 };
```

To generate output

```
.\tailwindcss-windows-x64.exe -i scripts/cssGen/input.css -o scripts/cssGen/output.css -c scripts/cssGen/tailwind.config.js
```
