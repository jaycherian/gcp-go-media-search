// Copyright 2024 Google, LLC
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     https://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {Box, Grid2, Typography, CircularProgress} from "@mui/material";
import {Scene} from "../shared/model";
import React, {useState, useEffect} from 'react';
import axios from 'axios';

const SceneData = ({mediaId, scene}: { mediaId: string, scene: Scene }) => {
    const [videoUrl, setVideoUrl] = useState<string | null>(null);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchSignedUrl = async () => {
            try {
                setLoading(true);
                setError(null);
                const response = await axios.get(`/api/v1/media/${mediaId}/stream`);
                setVideoUrl(response.data.url);
            } catch (err) {
                setError("Could not load video.");
                console.error("Error fetching signed URL:", err);
            } finally {
                setLoading(false);
            }
        };

        fetchSignedUrl();
    }, [mediaId]);


    const formatScript = (val: string): string => {
        return val.replace(/\n/g, "<br/>");
    }

    const GetStartTimeInSeconds = (): number => {
        const parts = scene.start.split(':');
        return parseInt(parts[0])*60*60 + parseInt(parts[1])*60 + parseInt(parts[2]);
    }

    const GetEndTimeInSeconds = (): number => {
        const parts = scene.end.split(':');
        return parseInt(parts[0])*60*60 + parseInt(parts[1])*60 + parseInt(parts[2]);
    }

    const videoSrc = videoUrl ? `${videoUrl}#t=${GetStartTimeInSeconds()},${GetEndTimeInSeconds()}` : '';

    return (
        <>
            <Grid2 size={6}>
                <Grid2 container spacing={2}>
                    <Grid2 size={4} sx={{fontWeight: 800}}>Sequence</Grid2>
                    <Grid2 size={4} sx={{fontWeight: 800}}>Start</Grid2>
                    <Grid2 size={4} sx={{fontWeight: 800}}>End</Grid2>

                    <Grid2 size={4}>{scene.sequence}</Grid2>
                    <Grid2 size={4}>{scene.start}</Grid2>
                    <Grid2 size={4}>{scene.end}</Grid2>
                </Grid2>
            </Grid2>
            <Grid2 size={6} >
                <Box sx={{display: 'flex', flex: 1, flexGrow: 1, justifyContent: 'center', justifyItems: 'center', alignItems: 'center', alignContent: 'center', padding: 2, minHeight: '150px'}}>
                {loading && <CircularProgress />}
                {error && <Typography color="error">{error}</Typography>}
                {videoUrl && (
                    <video controls style={{border: '1px solid #4285F4  ', borderRadius: '10px', boxShadow: '1px 1px 6px 1px #666', width: '100%'}}>
                        <source src={videoSrc} type="video/mp4" />
                        Your browser does not support the video tag.
                    </video>
                )}
                </Box>
            </Grid2>
            <Grid2 size={12} sx={{textAlign: 'left'}}><Typography component="div" variant="body2">
                <div dangerouslySetInnerHTML={{__html: formatScript(scene.script)}}/>
            </Typography></Grid2>
        </>
    )
};

export default SceneData
