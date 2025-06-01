#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  // Map Node.js platform/arch to our binary naming
  const platformMap = {
    'darwin': { x64: 'darwin-x64', arm64: 'darwin-arm64' },
    'linux': { x64: 'linux-x64', arm64: 'linux-arm64' },
    'win32': { x64: 'win32-x64', arm64: 'win32-arm64' }
  };

  if (!platformMap[platform] || !platformMap[platform][arch]) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return platformMap[platform][arch];
}

function installBinary() {
  try {
    const platformId = getPlatform();
    const packageName = `dtoforge-${platformId}`;

    console.log(`Installing DtoForge for ${platformId}...`);

    // Find the platform-specific package binary
    let binaryPath;
    try {
      // Look for the binary in the platform package
      const binaryName = process.platform === 'win32' ? 'dtoForge.exe' : 'dtoForge';
      binaryPath = require.resolve(`${packageName}/${binaryName}`);
    } catch (err) {
      throw new Error(`Platform package ${packageName} not found. This platform may not be supported.`);
    }

    // Create bin directory in the main package
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Copy binary to the main package's bin directory
    // npm will automatically create symlinks from node_modules/.bin/dtoforge to this file
    const targetPath = path.join(binDir, process.platform === 'win32' ? 'dtoForge.exe' : 'dtoForge');
    fs.copyFileSync(binaryPath, targetPath);

    // Make executable on Unix-like systems
    if (process.platform !== 'win32') {
      fs.chmodSync(targetPath, 0o755);
    }

    console.log(`✅ DtoForge installed successfully!`);
    console.log(`🎯 Binary available as: dtoforge`);
    console.log(`📦 Local usage: npx dtoforge --help`);
    console.log(`🌍 Global usage: dtoforge --help (if installed globally)`);

  } catch (error) {
    console.error(`❌ Installation failed: ${error.message}`);

    console.log(`
📥 Manual Installation Alternative:
Download the binary directly from GitHub releases:
https://github.com/eliraz-refael/dtoForge/releases/latest

Available platforms: Linux, macOS, Windows (x64, arm64)
`);

    process.exit(1);
  }
}

// Only run if this script is executed directly (not required)
if (require.main === module) {
  installBinary();
}

module.exports = { installBinary };
