import { defineConfig } from 'vite'

export default defineConfig({
  // Configure for Wails production builds
  base: './',
  build: {
    outDir: 'dist',
    rollupOptions: {
      // Don't bundle main.js - let it be copied as-is
      external: ['./src/main.js'],
      input: {
        main: './index.html'
      }
    },
    // Copy additional assets that Vite might not detect
    copyPublicDir: false, // We'll handle copying manually
  },
  // Plugin to copy main.js to the correct location
  plugins: [
    {
      name: 'copy-main-js',
      generateBundle() {
        // Copy main.js to the dist/src directory
        const fs = require('fs');
        const path = require('path');
        
        const srcDir = path.join(__dirname, 'dist', 'src');
        const srcFile = path.join(__dirname, 'src', 'main.js');
        const destFile = path.join(srcDir, 'main.js');
        
        if (!fs.existsSync(srcDir)) {
          fs.mkdirSync(srcDir, { recursive: true });
        }
        
        fs.copyFileSync(srcFile, destFile);
      }
    }
  ]
})