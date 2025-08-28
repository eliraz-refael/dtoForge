#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

function getPlatformBinary() {
  const platform = process.platform;
  const arch = process.arch;

  // Map Node.js platform/arch to our binary naming
  const platformMap = {
    'darwin': { x64: 'darwin-x64', arm64: 'darwin-arm64' },
    'linux': { x64: 'linux-x64', arm64: 'linux-arm64' },
    'win32': { x64: 'win32-x64.exe', arm64: 'win32-arm64.exe' }
  };

  if (!platformMap[platform] || !platformMap[platform][arch]) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return platformMap[platform][arch];
}

function installBinary() {
  try {
    const binaryName = getPlatformBinary();
    const sourcePath = path.join(__dirname, 'binaries', binaryName);

    console.log(`Installing DtoForge for ${process.platform}-${process.arch}...`);

    if (!fs.existsSync(sourcePath)) {
      throw new Error(`Binary for ${process.platform}-${process.arch} not found`);
    }

    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    if (process.platform === 'win32') {
      // On Windows, create the .exe file and a .cmd wrapper
      const targetPath = path.join(binDir, 'dtoforge.exe');
      const wrapperPath = path.join(binDir, 'dtoforge.cmd');
      
      fs.copyFileSync(sourcePath, targetPath);
      
      // Create .cmd wrapper for Windows
      const wrapperContent = `@echo off\n"%~dp0\\dtoforge.exe" %*\n`;
      fs.writeFileSync(wrapperPath, wrapperContent);
    } else {
      // On Unix-like systems, copy binary and create shell script wrapper
      const binaryPath = path.join(binDir, 'dtoforge-binary');
      const targetPath = path.join(binDir, 'dtoforge');
      
      fs.copyFileSync(sourcePath, binaryPath);
      fs.chmodSync(binaryPath, 0o755);
      
      // Create shell script wrapper
      const wrapperContent = `#!/bin/sh
basedir=$(dirname "$(echo "$0" | sed -e 's,\\\\,/,g')")

case \`uname\` in
    *CYGWIN*|*MINGW*|*MSYS*) basedir=\`cygpath -w "$basedir"\`;;
esac

exec "$basedir/dtoforge-binary" "$@"
`;
      fs.writeFileSync(targetPath, wrapperContent);
      fs.chmodSync(targetPath, 0o755);
    }

    console.log(`✅ DtoForge installed successfully!`);
    console.log(`🎯 Use: npx dtoforge --help`);
    console.log(`🌍 Or install globally: npm install -g dtoforge`);
    console.log(`📍 Platform: ${process.platform}-${process.arch}`);

  } catch (error) {
    console.error(`❌ Installation failed: ${error.message}`);
    console.log(`
📥 Manual Installation:
Download the binary directly from:
https://github.com/eliraz-refael/dtoForge/releases/latest
`);
    process.exit(1);
  }
}

// Only run if this script is executed directly (not required)
if (require.main === module) {
  installBinary();
}

module.exports = { installBinary };
