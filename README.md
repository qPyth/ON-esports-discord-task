# ON-esports-discord-task

## Description
>This bot records voice streams from a created voice channel.
The bot automatically connects to the channel that was created and begins to record the voices of all participants in the conversation.
The bot records audio from the channel until the time expires, which must be specified in the config, or leave 0, and then the recording will continue until there is no one left in the room.

## How to start
>You need to copy the contents of ***.env.example*** and paste all the necessary environment variables into ***.env*** and:
```bash
    go run .
```

## Features

> - The bot automatically connects to the created channel
> - In the configuration file, you can also configure the ***number of attempts*** to connect to the channel if the bot was unable to connect immediately
> - Recording is carried out until the ***time expires*** (if it is specified in the configuration file) or until the room remains empty
> - After stopping recording, the bot closes all connections and disconnects from the channel, and monitors the creation of new channels
> - After finishing recording, the bot sends records of all participants to AWS s3 in the format ***.ogg***
> - All records that for some reason were not sent to AWS s3 will not be deleted from the server and will be stored on it until further processing

## Commands

> Command in bot DM to get recordings of all participants for a specific voice channel: 
> ```text
> !records [voice channel ID]
> ```
> The bot will provide links to download files for the specified channel, or return messages that there are no entries for this channel

## My choices

> I decided to separate the implementation of voice recording into a separate layer. In its implementation, after connecting the bot to the channel, there were two triggers in separate goroutines to stop recording. This is a timer and periodic checking of users in the voice channel, when events come from one of two triggers, resources are stopped and released. And in a separate goroutine starting the process of sending records to S3.