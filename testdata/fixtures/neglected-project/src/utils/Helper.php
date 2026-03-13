<?php
// Legacy helper — duplicated logic (intentional for duplication testing)
// Known complexity: formatDate=4, sanitizeInput=5

namespace App\Utils;

class Helper
{
    // complexity: 1 (base) + 3 decision points = 4
    public static function formatDate($date, $format = 'Y-m-d')
    {
        if (empty($date)) {
            return '';
        }

        if (is_string($date)) {
            $timestamp = strtotime($date);
            if ($timestamp === false) {
                return $date;
            }
            return date($format, $timestamp);
        }

        return date($format, $date);
    }

    // complexity: 1 (base) + 4 decision points = 5
    public static function sanitizeInput($input)
    {
        if (is_array($input)) {
            $result = [];
            foreach ($input as $key => $value) {
                $result[$key] = self::sanitizeInput($value);
            }
            return $result;
        }

        if (!is_string($input)) {
            return $input;
        }

        $input = trim($input);
        if (strlen($input) > 10000) {
            $input = substr($input, 0, 10000);
        }

        return htmlspecialchars($input, ENT_QUOTES, 'UTF-8');
    }
}
