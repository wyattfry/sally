#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const manifestPath = path.join(__dirname, '../public/manifest.json');

// Get version from ./public/manifest.json 

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
const version = manifest.version;

// Increment version in ./public/manifest.json
const parts = version.split('.');
parts[2] = (parseInt(parts[2]) + 1).toString();
const newVersion = parts.join('.');
manifest.version = newVersion;
fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
console.log(`Version incremented from ${version} to ${newVersion}`);