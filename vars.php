<?php

    $offset = '';
    $dayFile = 'days'.$offset.'.txt';
    $dayLog = 'days'.$offset.'log.txt';
    $host = "192.168.56.10";

    $registry = '/tmp/registry.awfy';

    function error($msg) {
        //error_log("ALERT::".$msg."\n", 3, 'alert.log');
        error_log("ALERT::" . $msg . "\n");
    }

?>
