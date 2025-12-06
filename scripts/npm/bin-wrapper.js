#!/usr/bin/env node

/**
 * DtoForge npm bin wrapper
 *
 * This script ensures the platform-specific binary is installed
 * and then delegates execution to it. This approach is more robust
 * than relying on postinstall hooks which may not run in all scenarios.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Get the package root
// When used as a bin, __dirname is node_modules/.bin, so we need to find the actual package
// We resolve to the real path first to handle symlinks
const realPath = fs.realpathSync(__filename);
const packageRoot = path.dirname(realPath);
const binDir = path.join(packageRoot, 'bin');
const installScript = path.join(packageRoot, 'install.js');

function getBinaryPath() {
  if (process.platform === 'win32') {
    return path.join(binDir, 'dtoforge.exe');
  }
  return path.join(binDir, 'dtoforge-binary');
}

function ensureBinaryInstalled() {
  const binaryPath = getBinaryPath();

  // Check if binary exists
  if (fs.existsSync(binaryPath)) {
    return true;
  }

  // Binary not found, run installation
  console.log('DtoForge binary not found, installing...');

  try {
    // Run install.js synchronously - must call the exported function
    const { installBinary } = require(installScript);
    installBinary();
    return fs.existsSync(binaryPath);
  } catch (error) {
    console.error(`Failed to install DtoForge binary: ${error.message}`);
    return false;
  }
}

function main() {
  // Ensure binary is installed
  if (!ensureBinaryInstalled()) {
    console.error('❌ Could not install DtoForge binary for your platform.');
    console.log('\n📥 Manual Installation:');
    console.log('Download the binary directly from:');
    console.log('https://github.com/eliraz-refael/dtoForge/releases/latest');
    process.exit(1);
  }

  const binaryPath = getBinaryPath();

  // Spawn the binary with all arguments
  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: false,
  });

  child.on('error', (error) => {
    console.error(`Failed to execute DtoForge: ${error.message}`);
    process.exit(1);
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code || 0);
    }
  });
}

main();
