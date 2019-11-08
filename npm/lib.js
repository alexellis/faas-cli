const axios = require('axios');
const pkg = require('./package.json');
const fs = require('fs');
const { promisify } = require('util');
const exec = promisify(require('child_process').exec);

module.exports = {
    getSuffix(type, arch) {
        if (type === 'Windows_NT') {
            return '.exe';
        }

        if (type === 'Linux') {
            if (arch === 'x64') {
                return '';
            }

            if (arch === 'aarch64') {
                return '-arm64';
            }

            if (arch === 'armv61' || arch === 'armv71') {
                return '-armhf'
            }
        }

        if (type === 'Darwin') {
            return '-darwin';
        }

        throw new Error(`Unsupported platform: ${type} ${arch}`);
    },
    download(url, dest) {
        return new Promise(async (resolve, reject) => {
            let ws = fs.createWriteStream(dest);
            let res = await axios({ url, responseType: 'stream' });
            res.data
                .pipe(ws)
                .on('error', reject)
                .on('finish', () => {
                    resolve();
                });
        });
    },
    getReleases() {
        return axios.get(`https://api.github.com/repos/openfaas/faas-cli/releases/tags/${pkg.version}`)
            .then(res => res.data.assets);
    },
    async cmd(...args) {
        let { stdout, stderr } = await exec(...args);
        if (stdout) console.log(stdout);
        if (stderr) console.error(stderr);
    }
}