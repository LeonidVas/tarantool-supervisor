#! /usr/bin/env lua5.1

local signal = require('posix.signal')
local posix = require('posix.unistd')
local os = require('os')

--- Signal handler.
local function handler(signo)
    local ignore = os.getenv('INSTSIGIGNORE')
    if not(ignore and ignore:lower() == 'true') then
        os.exit(0)
    end
end

local function main()
    -- Set a signal handler.
    signal.signal(signal.SIGINT, handler)
    signal.signal(signal.SIGTERM, handler)

    while true do
        posix.sleep(1)
    end
end

main()
