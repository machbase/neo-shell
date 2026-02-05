'use strict';

const process = require('process');
const env = process.env;

const user = env.get('NEOSHELL_USER');
const password = env.get('NEOSHELL_PASSWORD');

env.set('NEO_USER', null);
env.set('NEO_PASSWORD', null);

process.exec('neo-shell')

console.println("disconnected from neo-shell");
env.set('NEO_USER', user);
env.set('NEO_PASSWORD', password);